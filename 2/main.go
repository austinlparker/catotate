package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/golang/freetype"

	"image/png"
	_ "image/png"

	"github.com/golang/freetype/truetype"
	"github.com/lightstep/lightstep-tracer-go"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
)

const catString = `
 /\_/\
( o.o )
 > ^ <
`

var (
	catAPIKey            = "53a92b01-302f-4e4b-85d9-1a2ec03d98dc"
	traceVerbose         = os.Getenv("TRACE_LEVEL") == "local"
	dpi          float64 = 72
	fontfile             = "../luximr.ttf"
	size         float64 = 48
	spacing              = 1.5
	font                 = buildFont()
)

func buildFont() *truetype.Font {
	fontBytes, err := ioutil.ReadFile(fontfile)
	if err != nil {
		panic("could not read fontfile")
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		panic("could not parse fontfile")
	}
	return f
}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/getHelloWorldCat", getCatHandler)
	mux.HandleFunc("/", index)

	mw := nethttp.Middleware(tracer, mux)
	log.Println("Server listening on port 3001")
	http.ListenAndServe(":3001", mw)
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, catString)
}

func getCatHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	catAPIResponse, err := getCatAPIResponse(r.Context())
	if err != nil {
		http.Error(w, "could not get cat api response", http.StatusInternalServerError)
	}
	log.Println(catAPIResponse)
	catImage, err := getCatImage(r.Context(), catAPIResponse.URL)
	if err != nil {
		http.Error(w, "could not get cat image", http.StatusInternalServerError)
	}
	annotationString := parseAnnotationString(r.Context(), r)
	catData, err := annotateCat(r.Context(), catImage, annotationString)
	if err != nil {
		http.Error(w, "could not annotate cat image", http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "%s", catData)
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
		ls.LogEvent(err.Error())
		ls.SetTag("error", true)
		finishLocalSpan(ls)
		return resObject, err
	}

	req = req.WithContext(ctx)
	req, ht := nethttp.TraceRequest(opentracing.GlobalTracer(), req)
	defer ht.Finish()

	res, err := client.Do(req)
	if err != nil {
		ls.LogEvent(err.Error())
		ls.SetTag("error", true)
		finishLocalSpan(ls)
		return resObject, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		ls.LogEvent(err.Error())
		ls.SetTag("error", true)
		finishLocalSpan(ls)
		return resObject, err
	}

	resObject = unmarshalAPIResponse(req.Context(), body)

	ht.Span().LogKV("response", string(body))
	ht.Span().LogKV("resource", resObject)

	res.Body.Close()
	finishLocalSpan(ls)
	return resObject, nil
}

func parseAnnotationString(ctx context.Context, r *http.Request) string {
	ctx, ls := localSpan(ctx)
	var annotationString string
	keys, ok := r.URL.Query()["annotation"]

	if !ok || len(keys[0]) < 1 {
		ls.LogEvent("could not find annotation param, falling back to default")
		annotationString = "hello world"
	} else {
		annotationString = keys[0]
	}

	finishLocalSpan(ls)
	return annotationString
}

func annotateCat(ctx context.Context, picture image.Image, s string) (string, error) {
	ctx, ls := localSpan(ctx)
	var c *freetype.Context

	switch picture := picture.(type) {
	case *image.RGBA:
		c = buildRGBAContext(ctx, picture)
	case *image.NRGBA:
		c = buildNRGBAContext(ctx, picture)
	}

	pt := freetype.Pt(10, 10+int(c.PointToFixed(size)>>6))
	_, err := c.DrawString(s, pt)
	if err != nil {
		ls.LogKV(err)
		ls.SetTag("error", true)
		finishLocalSpan(ls)
		return "", err
	}

	var outBuffer bytes.Buffer
	png.Encode(&outBuffer, picture)
	outEncodedString := base64.StdEncoding.EncodeToString(outBuffer.Bytes())
	finishLocalSpan(ls)
	return outEncodedString, nil
}

func buildRGBAContext(ctx context.Context, i *image.RGBA) *freetype.Context {
	ctx, ls := localSpan(ctx)
	fg := image.Black
	c := freetype.NewContext()
	c.SetDPI(dpi)
	c.SetFont(font)
	c.SetFontSize(size)
	c.SetClip(i.Bounds())
	c.SetDst(i)
	c.SetSrc(fg)
	finishLocalSpan(ls)
	return c
}

func buildNRGBAContext(ctx context.Context, i *image.NRGBA) *freetype.Context {
	ctx, ls := localSpan(ctx)
	fg := image.Black
	c := freetype.NewContext()
	c.SetDPI(dpi)
	c.SetFont(font)
	c.SetFontSize(size)
	c.SetClip(i.Bounds())
	c.SetDst(i)
	c.SetSrc(fg)
	finishLocalSpan(ls)
	return c
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
		span.LogKV(err)
		span.SetTag("error", true)
		finishLocalSpan(span)
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

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
