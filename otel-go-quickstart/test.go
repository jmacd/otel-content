package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"

	"go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/exporter/trace/jaeger"
	"go.opentelemetry.io/plugin/httptrace"
	sdk "go.opentelemetry.io/sdk/trace"
	"google.golang.org/grpc/codes"
)

func setupTracer() (trace.Tracer, *jaeger.Exporter, error) {
	// Register installs a new global tracer instance.
	tracer := sdk.Register()

	// Construct and register an export pipeline using the Jaeger
	// exporter and a span processor.
	exporter, err := jaeger.NewExporter(
		jaeger.Options{
			AgentEndpoint: "localhost:6831",
		},
	)
	if err != nil {
		return nil, nil, err
	}

	// A simple span processor calls through to the exporter
	// without buffering.
	ssp := sdk.NewSimpleSpanProcessor(exporter)
	sdk.RegisterSpanProcessor(ssp)

	// Use sdk.AlwaysSample sampler to send all spans.
	sdk.ApplyConfig(
		sdk.Config{
			DefaultSampler: sdk.AlwaysSample(),
		},
	)

	return tracer, exporter, nil
}

func main() {
	// Setup tracing and get a Tracer instance.  We'll use the
	// exporter to flush before exiting.
	tracer, exporter, err := setupTracer()

	if err != nil {
		log.Fatal("Could not initialize tracing: ", err)
	}

	// Tracing uses the standard context for propagation, we'll
	// start with a background context.
	ctx := context.Background()

	tracer.WithSpan(ctx, "foo",
		func(ctx context.Context) error {
			tracer.WithSpan(ctx, "bar",
				func(ctx context.Context) error {
					tracer.WithSpan(ctx, "baz",
						func(ctx context.Context) error {
							return nil
						},
					)
					return nil
				},
			)
			return nil
		},
	)

	// The Jaeger exporter will have buffered spans at this point, send them.
	exporter.Flush()
}

func sayHello() {
	ctx := context.Background()
	tracer := trace.GlobalTracer()

	ctx, trace := tracer.Start(ctx, "say-hello")

	trace.End()
}

func sayHello2() {
	ctx := context.Background()
	tracer := trace.GlobalTracer()

	err := tracer.WithSpan(ctx, "say-hello", func(ctx context.Context) error {
		// This body is traced, and the span will End() despite panics.
		return nil
	})

	if err != nil {
		// ...
	}
}

func sayHTTPHello(ctx context.Context) {
	var body []byte
	client := http.DefaultClient

	trace.GlobalTracer().WithSpan(ctx, "client-call",
		func(ctx context.Context) error {
			req, _ := http.NewRequest("GET", "http://localhost:7777/hello", nil)

			ctx, req = httptrace.W3C(ctx, req)
			httptrace.Inject(ctx, req)

			res, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			body, err = ioutil.ReadAll(res.Body)
			res.Body.Close()
			trace.CurrentSpan(ctx).SetStatus(codes.OK)

			return err
		})
}
