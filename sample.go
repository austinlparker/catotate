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