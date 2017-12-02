package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"io"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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

	span := ds.tracer.StartSpan(fmt.Sprintf("order_donut[%s]", p.Flavor), ext.RPCServerOption(clientContext))
	defer span.Finish()

	err := ds.makeDonut(span.Context(), p.Flavor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (ds *DonutService) handleState(w http.ResponseWriter, r *http.Request) {
	state := ds.state()
	data, err := json.Marshal(state)
	panicErr(err)
	w.Write(data)
}

func (ds *DonutService) webClean(w http.ResponseWriter, r *http.Request) {
	span := ds.tracer.StartSpan("cleaner")
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

	span := ds.tracer.StartSpan(
		fmt.Sprintf("restock[%s]", p.Flavor),
		ext.RPCServerOption(clientContext))

	ds.restock(span.Context(), p.Flavor)
}

func (ds *DonutService) state() *State {
	return &State{
		OilLevel:  ds.fryer.OilLevel(),
		Inventory: ds.inventory(),
	}
}

func (ds *DonutService) makeDonut(parentSpanContext opentracing.SpanContext, flavor string) error {
	donutSpan := ds.tracer.StartSpan("make_donut", ext.RPCServerOption(parentSpanContext))
	defer donutSpan.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), donutSpan)

	ds.payer.BuyDonut(ctx)
	ds.fryer.FryDonut(ctx)
	return ds.addTopping(donutSpan, flavor)
}

func (ds *DonutService) addTopping(span opentracing.Span, flavor string) error {
	ds.toppersLock.Lock(span)
	topper := ds.toppers[flavor]
	if topper == nil {
		topper = newTopper(ds.tracerGen, flavor, topDuration)
		ds.toppers[flavor] = topper
	}
	ds.toppersLock.Unlock()

	return topper.SprinkleTopping(opentracing.ContextWithSpan(context.Background(), span))
}

func (ds *DonutService) cleanFryer(parentSpanContext opentracing.SpanContext) {
	donutSpan := ds.tracer.StartSpan("clean_fryer", ext.RPCServerOption(parentSpanContext))
	defer donutSpan.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), donutSpan)

	ds.fryer.ChangeOil(ctx)
}

func (ds *DonutService) inventory() map[string]int {
	inventory := make(map[string]int)
	span := ds.tracer.StartSpan("inventory")
	defer span.Finish()

	ds.toppersLock.Lock(span)
	for flavor, topper := range ds.toppers {
		inventory[flavor] = topper.Quantity(span)
	}
	ds.toppersLock.Unlock()

	return inventory
}

func (ds *DonutService) restock(parentSpanContext opentracing.SpanContext, flavor string) {
	donutSpan := ds.tracer.StartSpan("restock_ingredients", ext.RPCServerOption(parentSpanContext))
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
