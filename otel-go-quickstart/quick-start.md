---
title: "Quick Start: Tracing"
weight: 1
---

The examples on this page use [OpenTelemetry-Go](https://github.com/open-telemetry/opentelemetry-go)(an OpenTelemetry comptible tracer). These examples assume that the [Jaeger all-in-one image](https://github.com/jaegertracing/jaeger) is running locally via Docker:

`$ docker run -d -p 6831:6831/udp -p 16686:16686 jaegertracing/all-in-one:latest`

## Setting up your tracer

There are three objects configured here: a new tracing SDK, a new
simple span processor, and a new Jaeger exporter.

```golang
import (
	"context"
	"log"

	"go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/exporter/trace/jaeger"
	sdk "go.opentelemetry.io/sdk/trace"
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
```

The returned `Tracer` is ready to use.  The returned `Exporter` will
be useful for flushing spans before exiting the process.

## Start and end a span

```golang
import "go.opentelemetry.io/api/trace"

func sayHello() {
	ctx := context.Background()
	tracer := trace.GlobalTracer()

	ctx, trace := tracer.Start(ctx, "say-hello")

	trace.End()
}
```

There is another way to start a span that supports recovering from
panics automatically, so that spans still send in these sitations.

```golang
import "go.opentelemetry.io/api/trace"

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
```

## Creating child spans

This example `main()` function sets up the tracer and sends three
spans, two of them children, as part of a single trace.  Note the call
to `exporter.Flush()` to send spans before exiting the process.

```golang
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

	_ = tracer.WithSpan(ctx, "foo",
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
```

## Trace an HTTP request

This example shows how to propagate trace context across an HTTP
request.  First we have to select the type of context carrier--in this
example we use W3C.  On the client side, a call to `httptrace.W3C`
configures the `http.Request` object to use W3C trace context headers,
and a call to `httptrace.Inject` both injects the headers and applies
conventional HTTP attributes to the span.

```golang
import (
	"context"
	"io/ioutil"
	"net/http"

	"go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/plugin/httptrace"
	"google.golang.org/grpc/codes"
)

func sayHTTPHello(ctx context.Context) {
	var body []byte
	client := http.DefaultClient
	tracer := trace.GlobalTracer()

	tracer.WithSpan(ctx, "client-call",
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
```

Continuing on the server side, this is an HTTP handler that adds
tracing support.

```golang
func helloHandler(w http.ResponseWriter, req *http.Request) {
	tracer := trace.GlobalTracer()

	// Extracts the conventional HTTP span attributes,
	// distributed context tags, and a span context for
	// tracing this request.
	attrs, tags, spanCtx := httptrace.Extract(req.Context(), req)

	// Apply the distributed context tags to the request
	// context.
	req = req.WithContext(tag.WithMap(req.Context(), tag.NewMap(tag.MapUpdate{
		MultiKV: tags,
	})))

	// Start the server-side span, passing the remote
	// child span context explicitly.
	_, span := tracer.Start(
		req.Context(),
		"hello",
		trace.WithAttributes(attrs...),
		trace.ChildOf(spanCtx),
	)
	defer span.End()

	_, _ = io.WriteString(w, "Hello, world!\n")
}
```

## View your traces

If you have Jaeger all-in-one running, you can view your traces at `localhost:16686`.
