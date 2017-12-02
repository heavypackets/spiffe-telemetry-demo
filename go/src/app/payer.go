package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	opentracing "github.com/opentracing/opentracing-go"
)

type Payer struct {
	tracer   opentracing.Tracer
	duration time.Duration
}

func NewPayer(tracerGen TracerGenerator, duration time.Duration) *Payer {
	return &Payer{
		tracer:   tracerGen("charge-hard"),
		duration: duration,
	}
}

func (m *Payer) BuyDonut(ctx context.Context) {
	var parentSpanContext opentracing.SpanContext
	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentSpanContext = parent.Context()
	}
	span := m.tracer.StartSpan("process_payment", opentracing.ChildOf(parentSpanContext))
	defer span.Finish()

	span.LogEvent(fmt.Sprint("Order: ", span.BaggageItem(donutOriginKey)))
	SleepGaussian(m.duration, 1)
}
