package telemetry

import (
	"context"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const peerTransientMapPrefix = "peer"

type TraceContext struct {
	ctx       context.Context
	remote    bool
	remoteCtx context.Context
}

type TracingHandler struct {
	Tracer      trace.Tracer
	Propagators propagation.TextMapPropagator
	isInit      bool
}

// TracingIsInit checks if telemetry was initialized
func (th *TracingHandler) TracingIsInit() bool {
	return th.isInit
}

// TracingInit sets tracing telemetry init param as true
func (th *TracingHandler) TracingInit() {
	th.isInit = true
}

// StartNewSpan starts new span
func (th *TracingHandler) StartNewSpan(traceCtx TraceContext, spanName string, opts ...trace.SpanStartOption) (TraceContext, trace.Span) {
	if traceCtx.ctx == nil {
		traceCtx.ctx = context.Background()
	}

	ctx, span := th.Tracer.Start(traceCtx.ctx, spanName, opts...)
	return TraceContext{
		ctx:       ctx,
		remote:    traceCtx.remote,
		remoteCtx: traceCtx.remoteCtx,
	}, span
}

func (th *TracingHandler) ContextFromStub(stub shim.ChaincodeStubInterface) TraceContext {
	traceCtx := TraceContext{
		ctx: th.Propagators.Extract(context.Background(), propagation.MapCarrier{}),
	}

	transientMap, err := stub.GetTransient()
	if err != nil {
		return traceCtx
	}

	carrier, err := UnpackTransientMap(transientMap)
	if err != nil {
		return traceCtx
	}

	peerTransientMap := make(map[string][]byte)
	for k, v := range transientMap {
		if strings.HasPrefix(k, peerTransientMapPrefix) {
			keyWithoutPrefix := strings.TrimPrefix(k, peerTransientMapPrefix)
			peerTransientMap[keyWithoutPrefix] = v
		}
	}

	if len(peerTransientMap) != 0 {
		peerCarrier, err := UnpackTransientMap(peerTransientMap)
		if err != nil {
			return traceCtx
		}

		traceParentContext := th.Propagators.Extract(context.Background(), carrier)
		traceCtx.ctx = th.Propagators.Extract(context.Background(), peerCarrier)
		traceCtx.remote = trace.SpanContextFromContext(traceParentContext).IsRemote()
		traceCtx.remoteCtx = traceParentContext
		return traceCtx
	}

	traceCtx.ctx = th.Propagators.Extract(context.Background(), carrier)
	traceCtx.remote = trace.SpanContextFromContext(traceCtx.ctx).IsRemote()
	traceCtx.remoteCtx = traceCtx.ctx

	return traceCtx
}

func (th *TracingHandler) RemoteCarrier(traceCtx TraceContext) propagation.MapCarrier {
	carrier := propagation.MapCarrier{}
	if !traceCtx.remote {
		return carrier
	}

	th.Propagators.Inject(traceCtx.remoteCtx, carrier)
	return carrier
}

func (th *TracingHandler) ExtractContext(carrier propagation.MapCarrier) TraceContext {
	return TraceContext{
		ctx: th.Propagators.Extract(context.Background(), carrier),
	}
}
