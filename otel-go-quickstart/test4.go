package gointro

import (
	"io"
	"net/http"

	"go.opentelemetry.io/api/tag"
	"go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/plugin/httptrace"
)

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
