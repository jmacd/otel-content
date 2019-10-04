package gointro

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
