package main
 
import (
  "context"
  "flag"
  "fmt"
  "math"
  "net/http"
  _ "net/http/pprof"
  "time"
 
  lightstep "github.com/lightstep/lightstep-tracer-go"
  opentracing "github.com/opentracing/opentracing-go"
  jaeger "github.com/uber/jaeger-client-go"
  "github.com/uber/jaeger-client-go/config"
  "github.com/uber/jaeger-client-go/log"
  "github.com/uber/jaeger-lib/metrics"
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
  tracerType         = flag.String("tracer_type", "lightstep", "")
  orderProcesses     = flag.Int("order", 1, "")
  restockerProcesses = flag.Int("restock", 0, "")
  cleanerProcesses   = flag.Int("clean", 0, "")
)
 
func SleepGaussian(d time.Duration, queueLength float64) {
  cappedDuration := float64(d)
  if queueLength > 4 {
    cappedDuration = math.Min(float64(time.Millisecond*50), maxQueueDuration/(queueLength-4))
  }
  //  noise := (float64(cappedDuration) / 3) * rand.NormFloat64()
  time.Sleep(time.Duration(cappedDuration))
}
 
type TracerGenerator func(component string) opentracing.Tracer
 
func main() {
  flag.Parse()
  lightstep.SetGlobalEventHandler(lightstep.NewEventLogOneError())
  var tracerGen TracerGenerator
  if *tracerType == "lightstep" {
    tracerGen = func(component string) opentracing.Tracer {
      return lightstep.NewTracer(lightstep.Options{
        AccessToken: *accessToken,
        Collector: lightstep.Endpoint{
          Host: *collectorHost,
          Port: *collectorPort,
        },
        MaxBufferedSpans: 200000,
        UseGRPC:          true,
        Tags: opentracing.Tags{
          lightstep.ComponentNameKey: component,
        },
      })
    }
  } else if *tracerType == "jaeger" {
    cfg := config.Configuration{
      Sampler: &jaegercfg.SamplerConfig{
        Type:  jaeger.SamplerTypeConst,
        Param: 1,
      },
    }
    closer, err := cfg.InitGlobalTracer(
      component,
      config.Logger(log.StdLogger),