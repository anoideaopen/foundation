package telemetry

import (
	"encoding/json"
	"go.opentelemetry.io/otel/propagation"
)

const (
	tracingKey     = "tracing_peer"
	peerTracingKey = "peer_trace_id"
)

// PackToTransientMap prepares carrier for using in transient map
func PackToTransientMap(traceCarrier propagation.MapCarrier) (map[string][]byte, error) {
	transientMap := make(map[string][]byte)
	for _, k := range traceCarrier.Keys() {
		rawValue := []byte(traceCarrier.Get(k))
		transientMap[k] = rawValue
	}

	return transientMap, nil
}

// GetCarriersFromTransientMap getting carriers from transient map values by keys 'tracing_peer' or 'peer_trace_id'
func GetCarriersFromTransientMap(transientMap map[string][]byte) (propagation.MapCarrier, propagation.MapCarrier, error) {
	var traceCarrier propagation.MapCarrier
	var tracePeerCarrier propagation.MapCarrier
	for k, v := range transientMap {
		if k == tracingKey {
			mc := propagation.MapCarrier{}
			if err := json.Unmarshal(v, &mc); err != nil {
				return nil, nil, err
			}
			traceCarrier = mc
		}
		if k == peerTracingKey {
			mc := propagation.MapCarrier{}
			if err := json.Unmarshal(v, &mc); err != nil {
				return nil, nil, err
			}
			tracePeerCarrier = mc
		}

	}

	return traceCarrier, tracePeerCarrier, nil
}
