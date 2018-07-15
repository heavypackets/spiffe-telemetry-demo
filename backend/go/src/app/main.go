package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"

	spiffe "github.com/spiffe/go-spiffe/uri"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	donutOriginKey   = "origin"
	maxQueueDuration = float64(8 * time.Second)
)

var (
	baseDir            = flag.String("basedir", "/etc/app/", "")
	accessToken        = flag.String("token", "DEVELOPMENT_TOKEN_bhs", "")
	port               = flag.Int("port", 8080, "")
	serviceHostport    = flag.String("service_hostport", "localhost:8080", "")
	collectorHost      = flag.String("collector_host", "collector-grpc.lightstep.com", "")
	collectorPort      = flag.Int("collector_port", 443, "")
	tracerType         = flag.String("tracer_type", "jaeger", "")
	orderProcesses     = flag.Int("order", 1, "")
	restockerProcesses = flag.Int("restock", 0, "")
	cleanerProcesses   = flag.Int("clean", 0, "")
	svidPem            = flag.String("svid", "/certs/svid.pem", "")
	svid               = ""
)

func SleepGaussian(d time.Duration, queueLength float64) {
	cappedDuration := float64(d)
	if queueLength > 4 {
		cappedDuration = math.Min(float64(time.Millisecond*50), maxQueueDuration/(queueLength-4))
	}
	//	noise := (float64(cappedDuration) / 3) * rand.NormFloat64()
	time.Sleep(time.Duration(cappedDuration))
}

type TracerGenerator func(component string) opentracing.Tracer

func main() {
	flag.Parse()

	for {
		if err := parseSVID(*svidPem); err != nil {
			time.Sleep(time.Duration(time.Second * 3))

			continue
		}

		break
	}

	var tracerGen TracerGenerator
	if *tracerType == "jaeger" {
		tracerGen = func(component string) opentracing.Tracer {
			cfg := config.Configuration{
				Sampler: &config.SamplerConfig{
					Type:  jaeger.SamplerTypeConst,
					Param: 1,
				},
				Reporter: &config.ReporterConfig{
					LocalAgentHostPort: fmt.Sprintf("%s:6831", os.Getenv("JAEGER_AGENT_HOST")),
				},
			}
			tracer, _, err := cfg.New(component)
			if err != nil {
				panic(err)
			}
			return tracer
		}
	} else {
		panic(*tracerType)
	}
	ds := newDonutService(tracerGen)
	setupTelemetry(ds)

	// Make fake queries in the background.
	//	backgroundProcess(*orderProcesses, ds, runFakeUser)
	//	backgroundProcess(*restockerProcesses, ds, runFakeRestocker)
	//	backgroundProcess(*cleanerProcesses, ds, runFakeCleaner)

	http.HandleFunc("/", ds.pageHandler("order"))
	http.HandleFunc("/clean", ds.pageHandler("clean"))
	http.HandleFunc("/restock", ds.pageHandler("restock"))

	http.HandleFunc("/api/clean", ds.webClean)
	http.HandleFunc("/api/order", ds.webOrder)
	http.HandleFunc("/api/restock", ds.webRestock)

	http.HandleFunc("/service/fry", ds.serviceFry)
	http.HandleFunc("/service/top", ds.serviceTop)

	http.HandleFunc("/status", ds.handleState)

	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir(*baseDir+"public/"))))

	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Starting on :", *port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	fmt.Println("Exiting", err)
}

func backgroundProcess(max int, ds *DonutService, f func(flavor string, ds *DonutService)) {
	for i := 0; i < max; i++ {
		var flavor string
		switch i % 3 {
		case 0:
			flavor = "cinnamon"
		case 1:
			flavor = "chocolate"
		case 2:
			flavor = "sprinkles"
		}
		go f(flavor, ds)
	}
}

func runFakeUser(flavor string, ds *DonutService) {
	for {
		SleepGaussian(2500*time.Millisecond, 1)
		span := startSpan(fmt.Sprintf("background_order[%s]", flavor), ds.tracer)
		ds.makeDonut(span.Context(), flavor)
		span.Finish()
	}
}

func runFakeRestocker(flavor string, ds *DonutService) {
	for {
		SleepGaussian(20000*time.Millisecond, 1)
		span := startSpan(fmt.Sprintf("background_restocker[%s]", flavor), ds.tracer)
		ds.restock(span.Context(), flavor)
		span.Finish()
	}
}

func runFakeCleaner(flavor string, ds *DonutService) {
	for {
		SleepGaussian(time.Second, 1)
		span := startSpan("background_cleaner", ds.tracer)
		ds.cleanFryer(span.Context())
		span.Finish()
	}
}

func setSpanSVID(span opentracing.Span) {
	span.SetTag("spiffe_id", svid)
}

func startSpan(name string, tracer opentracing.Tracer, opts ...opentracing.StartSpanOption) opentracing.Span {
	span := tracer.StartSpan(name, opts...)
	setSpanSVID(span)

	return span
}

func startSpanFronContext(name string, tracer opentracing.Tracer, ctx context.Context) opentracing.Span {
	var parentSpanContext opentracing.SpanContext
	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentSpanContext = parent.Context()
		setSpanSVID(parent)
	}
	return startSpan(name, tracer, opentracing.ChildOf(parentSpanContext))
}

func parseSVID(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	uris, err := spiffe.FGetURINamesFromPEM(f)
	_ = f.Close()

	if err != nil {
		return err
	}

	if len(uris) == 0 {
		return errors.New("No URIs found in specified SVID pem")
	}

	// Assume only one URI per SPIFFE spec
	svid = uris[0]
	fmt.Printf("SPIFFE ID [%s] verified\n", svid)

	return nil
}

func setupTelemetry(ds *DonutService) error {
	ds.totalOrderedDonuts = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "donutshop_total_ordered_donuts",
		Help: "Number of donuts ordered.",
	})
	if err := prometheus.Register(ds.totalOrderedDonuts); err != nil {
		return err
	}

	ds.orderedDonuts = make(map[string]prometheus.Counter)
	ds.donutStock = make(map[string]prometheus.Gauge)

	return nil
}
