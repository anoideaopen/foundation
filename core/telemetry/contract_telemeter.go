package telemetry

import (
	"context"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

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

	// Getting decorators from the peer.
	decorators := stub.GetDecorations()
	// Unpacking decorators to carrier map.
	peerCarrier, err := UnpackTransientMap(decorators)
	if err != nil {
		return traceCtx
	}

	transientMap, err := stub.GetTransient()
	if err != nil {
		return traceCtx
	}

	carrier, err := UnpackTransientMap(transientMap)
	if err != nil {
		return traceCtx
	}

	traceCtx.ctx = th.Propagators.Extract(context.Background(), carrier)
	traceCtx.remote = trace.SpanContextFromContext(traceCtx.ctx).IsRemote()
	traceCtx.remoteCtx = traceCtx.ctx

	// If the carrier is not empty, it may indicate that a parent span context was received from the peer's decorator.
	if len(peerCarrier) != 0 {
		// The carrier from the peer's decorators may contain something other than trace parameters
		// Therefore, we first check that the result of Extract is not an empty context
		// and only in this case do we set it as the parent context.
		ctx := th.Propagators.Extract(context.Background(), peerCarrier)
		if ctx != context.Background() {
			traceCtx.ctx = th.Propagators.Extract(context.Background(), peerCarrier)
		}
	}
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
