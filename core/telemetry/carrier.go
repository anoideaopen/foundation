package telemetry

import (
	"go.opentelemetry.io/otel/propagation"
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

// UnpackTransientMap unpacks transient map into carrier
func UnpackTransientMap(transientMap map[string][]byte) (propagation.MapCarrier, error) {
	traceCarrier := propagation.MapCarrier{}
	for k, v := range transientMap {
		traceCarrier.Set(k, string(v))
	}

	return traceCarrier, nil
}
