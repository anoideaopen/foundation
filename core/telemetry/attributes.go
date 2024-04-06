package telemetry

import "go.opentelemetry.io/otel/attribute"

type MethodTypeNum int

func (t MethodTypeNum) String() string {
	switch t {
	case MethodQuery:
		return "query"
	case MethodTx:
		return "tx"
	case MethodNbTx:
		return "nbtx"
	case MethodUnknown:
		fallthrough
	default:
		return "unknown"
	}
}

const (
	MethodUnknown MethodTypeNum = iota
	MethodQuery
	MethodTx
	MethodNbTx
)

func MethodType(t MethodTypeNum) attribute.KeyValue {
	return attribute.String("method_type", t.String())
}
