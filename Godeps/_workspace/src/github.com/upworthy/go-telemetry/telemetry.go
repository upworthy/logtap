package telemetry

import (
	"encoding/json"
	"fmt"
	"log"
)

// Metrics is the interface that reports metrics to some telemetry
// receiver.
type Metrics interface {
	Value(value interface{}, name string, props ...interface{})
	Count(value int, name string, props ...interface{})
}

type discardMetrics struct{}

func (*discardMetrics) Value(value interface{}, name string, props ...interface{}) {}
func (*discardMetrics) Count(value int, name string, props ...interface{})         {}

// Discard discards telemetry data.
var Discard = &discardMetrics{}

type logMetrics struct{}

func printAppMetric(key string, value interface{}, name string) {
	var v, n []byte
	switch value.(type) {
	case float32, float64,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		v, _ = json.Marshal(value)
	default:
		return
	}
	n, _ = json.Marshal(name)
	log.Print(fmt.Sprintf(`APP_METRIC {"stat": %s, "%s": %s}`, n, key, v))
}

func (*logMetrics) Value(value interface{}, name string, props ...interface{}) {
	printAppMetric("value", value, name)
}

func (*logMetrics) Count(value int, name string, props ...interface{}) {
	printAppMetric("count", value, name)
}

// LogMetrics reports metrics to the standard logger.
//
// Examples:
//
//     LogMetrics.Count(33, "beans")  // prints `APP_METRIC {"stat": "beans", "count": 33}`
//     LogMetrics.Value(36.6, "temp") // prints `APP_METRIC {"stat": "temp", "value": 36.6}`
var LogMetrics = &logMetrics{}
