package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"io"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	fryDuration = time.Millisecond * 50
	payDuration = time.Millisecond * 250
	topDuration = time.Millisecond * 350
)

type State struct {
	OilLevel  int
	Inventory map[string]int
}

type DonutService struct {
	tracer    opentracing.Tracer
	payer     *Payer
	fryer     *Fryer
	tracerGen TracerGenerator

	totalOrderedDonuts prometheus.Counter
	orderedDonuts      map[string]prometheus.Counter
	donutStock         map[string]prometheus.Gauge

	toppersLock *SmartLock
	toppers     map[string]*Topper
}

func newDonutService(tracerGen TracerGenerator) *DonutService {
	return &DonutService{
		tracer:      tracerGen("donut-webserver"),
		payer:       NewPayer(tracerGen, payDuration),
		fryer:       newFryer(tracerGen, fryDuration),
		toppers:     make(map[string]*Topper),
		toppersLock: NewSmartLock(true),
		tracerGen:   tracerGen,
	}
}

func (ds *DonutService) pageHandler(pageBasename string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := template.New("").ParseFiles(
			*baseDir+pageBasename+".go.html",
			*baseDir+"header.go.html",
			*baseDir+"status.go.html")
		panicErr(err)

		err = t.ExecuteTemplate(w, pageBasename+".go.html", ds.state())
		panicErr(err)
	}
}

func (ds *DonutService) webOrder(w http.ResponseWriter, r *http.Request) {
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	clientContext, _ := ds.tracer.Extract(opentracing.HTTPHeaders, carrier)

	p := struct {
		Flavor string `json:"flavor"`
	}{}
	unmarshalJSON(r.Body, &p)
	if p.Flavor == "" {
		panic("flavor not set")
	}

	span := startSpan(fmt.Sprintf("order_donut[%s]", p.Flavor), ds.tracer, opentracing.ChildOf(clientContext))
	defer span.Finish()

	err := ds.makeDonut(span.Context(), p.Flavor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ds.totalOrderedDonuts.Inc()
	if c := ds.orderedDonuts[p.Flavor]; c != nil {
		c.Inc()
	}
}

func (ds *DonutService) handleState(w http.ResponseWriter, r *http.Request) {
	state := ds.state()
	data, err := json.Marshal(state)
	panicErr(err)
	w.Write(data)
}

func (ds *DonutService) webClean(w http.ResponseWriter, r *http.Request) {
	span := startSpan("cleaner", ds.tracer)
	ds.cleanFryer(span.Context())
}

func (ds *DonutService) webRestock(w http.ResponseWriter, r *http.Request) {
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	clientContext, _ := ds.tracer.Extract(opentracing.HTTPHeaders, carrier)

	p := struct {
		Flavor string `json:"flavor"`
	}{}
	unmarshalJSON(r.Body, &p)
	if p.Flavor == "" {
		panic("flavor not set")
	}

	span := startSpan(
		fmt.Sprintf("restock[%s]", p.Flavor), ds.tracer,
		opentracing.ChildOf(clientContext))

	ds.restock(span.Context(), p.Flavor)
}

func (ds *DonutService) serviceFry(w http.ResponseWriter, r *http.Request) {
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	clientContext, _ := ds.tracer.Extract(opentracing.HTTPHeaders, carrier)

	span := startSpan(
		"fry", ds.tracer,
		opentracing.ChildOf(clientContext))
	defer span.Finish()

	goCtx := opentracing.ContextWithSpan(context.Background(), span)
	ds.fryer.FryDonut(goCtx)
}

func (ds *DonutService) serviceTop(w http.ResponseWriter, r *http.Request) {
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	clientContext, _ := ds.tracer.Extract(opentracing.HTTPHeaders, carrier)

	p := struct {
		Flavor string `json:"flavor"`
	}{}
	unmarshalJSON(r.Body, &p)
	if p.Flavor == "" {
		panic("flavor not set")
	}

	span := startSpan(
		"top", ds.tracer,
		opentracing.ChildOf(clientContext))
	defer span.Finish()

	err := ds.addTopping(span, p.Flavor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusGone)
	}
}

func (ds *DonutService) state() *State {
	return &State{
		OilLevel:  ds.fryer.OilLevel(),
		Inventory: ds.inventory(),
	}
}

func (ds *DonutService) call(clientSpanContext opentracing.SpanContext, path string, postBody []byte) error {
	url := fmt.Sprintf("http://%s%s", *serviceHostport, path)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postBody))
	req.Header.Set("Content-Type", "application/json")
	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	err = ds.tracer.Inject(clientSpanContext, opentracing.HTTPHeaders, carrier)
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("call failure")
	}
	return nil
}

func (ds *DonutService) makeDonut(parentSpanContext opentracing.SpanContext, flavor string) error {
	donutSpan := startSpan("make_donut", ds.tracer, opentracing.ChildOf(parentSpanContext))
	defer donutSpan.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), donutSpan)

	ds.payer.BuyDonut(ctx)
	err := ds.call(donutSpan.Context(), "/service/fry", []byte{})
	if err != nil {
		return err
	}
	return ds.call(
		donutSpan.Context(),
		"/service/top",
		[]byte(fmt.Sprintf(`{"flavor":"%s"}`, flavor)))
}

func (ds *DonutService) addTopping(span opentracing.Span, flavor string) error {
	ds.toppersLock.Lock(span)
	defer ds.toppersLock.Unlock()

	topper := ds.toppers[flavor]
	if topper == nil {
		topper = newTopper(ds.tracerGen, flavor, topDuration)
		topper.ds = ds
		ds.toppers[flavor] = topper
		setupToppingTelemetry(topper, ds)
	}

	return topper.SprinkleTopping(opentracing.ContextWithSpan(context.Background(), span))
}

func setupToppingTelemetry(t *Topper, ds *DonutService) error {
	flavor := t.donutType
	name := fmt.Sprintf("donutshop_ordered_%s_donuts", flavor)
	help := fmt.Sprintf("Number of %s donuts ordered.", flavor)

	ds.orderedDonuts[flavor] = prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
	if err := prometheus.Register(ds.orderedDonuts[flavor]); err != nil {
		return err
	}

	name = fmt.Sprintf("donutshop_%s_donuts_stock", flavor)
	help = fmt.Sprintf("Number of %s donuts in stock.", flavor)

	ds.donutStock[flavor] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})
	if err := prometheus.Register(ds.donutStock[flavor]); err != nil {
		return err
	}
	ds.donutStock[flavor].Set(float64(t.quantity))

	return nil
}

func (ds *DonutService) cleanFryer(parentSpanContext opentracing.SpanContext) {
	donutSpan := startSpan("clean_fryer", ds.tracer, opentracing.ChildOf(parentSpanContext))
	defer donutSpan.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), donutSpan)

	ds.fryer.ChangeOil(ctx)
}

func (ds *DonutService) inventory() map[string]int {
	inventory := make(map[string]int)
	span := startSpan("inventory", ds.tracer)
	defer span.Finish()

	ds.toppersLock.Lock(span)
	for flavor, topper := range ds.toppers {
		inventory[flavor] = topper.Quantity(span)
	}
	ds.toppersLock.Unlock()

	return inventory
}

func (ds *DonutService) restock(parentSpanContext opentracing.SpanContext, flavor string) {
	donutSpan := startSpan("restock_ingredients", ds.tracer, opentracing.ChildOf(parentSpanContext))
	defer donutSpan.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), donutSpan)

	ds.toppersLock.Lock(donutSpan)
	topper := ds.toppers[flavor]
	if topper == nil {
		topper = newTopper(ds.tracerGen, flavor, topDuration)
		ds.toppers[flavor] = topper
	}
	ds.toppersLock.Unlock()

	topper.Restock(ctx)
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func unmarshalJSON(body io.ReadCloser, data interface{}) {
	defer body.Close()
	decoder := json.NewDecoder(body)
	err := decoder.Decode(&data)
	panicErr(err)
}
