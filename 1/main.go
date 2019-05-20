package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/lightstep/lightstep-tracer-go"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"

	_ "image/png"
)

var (
	serverPort   = ":3001"
	catAPIKey    = "53a92b01-302f-4e4b-85d9-1a2ec03d98dc"
	traceVerbose = os.Getenv("TRACE_LEVEL") == "local"
)

func main() {
	tracer := lightstep.NewTracer(lightstep.Options{
		Collector: lightstep.Endpoint{
			Host:      "localhost",
			Port:      8360,
			Plaintext: true,
		},
		AccessToken: "dev",
	})
	opentracing.SetGlobalTracer(tracer)
	fs := http.FileServer(http.Dir("../static"))
	mux := http.NewServeMux()
	mux.HandleFunc("/annotateCat", getCatHandler)
	mux.Handle("/", fs)

	mw := nethttp.Middleware(tracer, mux)
	log.Printf("Server listening on port %s", serverPort)
	http.ListenAndServe(serverPort, mw)
}

func getCatHandler(w http.ResponseWriter, r *http.Request) {
	catAPIResponse, err := getCatAPIResponse(r.Context())
	if err != nil {
		http.Error(w, "could not get cat api response", http.StatusInternalServerError)
	}
	catImage, err := getCatImage(r.Context(), catAPIResponse.URL)
	if err != nil {
		http.Error(w, "could not get cat image", http.StatusInternalServerError)
	}
	log.Println(catImage)
	w.WriteHeader(http.StatusOK)
}

func getCatImage(ctx context.Context, url string) (image.Image, error) {
	ctx, ls := localSpan(ctx)
	ls.LogEvent(fmt.Sprintf("getting cat image at %s", url))
	var imgData image.Image
	client := &http.Client{Transport: &nethttp.Transport{}}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("x-api-key", catAPIKey)
	if err != nil {
		ls.LogEvent("failed to build request")
		finishLocalSpan(ls)
		return imgData, err
	}

	req = req.WithContext(ctx)
	req, ht := nethttp.TraceRequest(opentracing.GlobalTracer(), req)
	defer ht.Finish()

	res, err := client.Do(req)
	if err != nil {
		ls.LogEvent("failed to talk to external service")
		finishLocalSpan(ls)
		return imgData, err
	}

	imgData, _, err = image.Decode(res.Body)
	if err != nil {
		ls.LogKV(err)
		ls.LogEvent("failed to decode body")
		finishLocalSpan(ls)
		return imgData, err
	}
	res.Body.Close()
	finishLocalSpan(ls)
	return imgData, nil
}

func getCatAPIResponse(ctx context.Context) (CatAPIResponse, error) {
	ctx, ls := localSpan(ctx)
	var resObject CatAPIResponse

	client := &http.Client{Transport: &nethttp.Transport{}}
	req, err := http.NewRequest("GET", "https://api.thecatapi.com/v1/images/search?mime_types=png", nil)
	req.Header.Set("x-api-key", catAPIKey)
	if err != nil {
		return resObject, err
	}

	req = req.WithContext(ctx)
	req, ht := nethttp.TraceRequest(opentracing.GlobalTracer(), req)
	defer ht.Finish()

	res, err := client.Do(req)
	if err != nil {
		return resObject, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return resObject, err
	}

	resObject = unmarshalAPIResponse(req.Context(), body)

	ht.Span().LogKV("response", string(body))
	ht.Span().LogKV("resource", resObject)

	res.Body.Close()
	finishLocalSpan(ls)
	return resObject, nil
}

func annotateCat() {

}

//CatAPIResponse is a cat api response with minimum required fields parsed out
type CatAPIResponse struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func unmarshalAPIResponse(ctx context.Context, b []byte) CatAPIResponse {
	ctx, span := localSpan(ctx)
	defer finishLocalSpan(span)

	var f []CatAPIResponse
	err := json.Unmarshal(b, &f)
	if err != nil {
		panic(err)
	}
	return f[0]
}

func localSpan(ctx context.Context) (context.Context, opentracing.Span) {
	if traceVerbose {
		pc, _, _, ok := runtime.Caller(1)
		fnCaller := runtime.FuncForPC(pc)
		if ok && fnCaller != nil {
			span, ctx := opentracing.StartSpanFromContext(ctx, fnCaller.Name())
			return ctx, span
		}
	}
	return ctx, opentracing.SpanFromContext(ctx)
}

func finishLocalSpan(span opentracing.Span) {
	if traceVerbose {
		span.Finish()
	}
}
