package logtap

import (
	"encoding/json"
	"fmt"
	"log"
)

// Telemetry is the interface that represents a telemetry receiver.
type Telemetry interface {
	Value(value interface{}, name string)
	Count(value int, name string)
}

type discardTelemetry struct{}

// DiscardTelemetry discards telemetry data.
var DiscardTelemetry = &discardTelemetry{}

func (*discardTelemetry) Value(value interface{}, name string) {}
func (*discardTelemetry) Count(value int, name string)         {}

type logTelemetry struct{}

// LogTelemetry sends telemetry to the standard logger.
//
// Examples:
//
//     LogTelemetry.Count(33, "beans")  // prints `APP_METRIC {"stat": "beans", "count": 33}`
//     LogTelemetry.Value(36.6, "temp") // prints `APP_METRIC {"stat": "temp", "value": 36.6}`
var LogTelemetry = &logTelemetry{}

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

func (*logTelemetry) Value(value interface{}, name string) {
	printAppMetric("value", value, name)
}

func (*logTelemetry) Count(value int, name string) {
	printAppMetric("count", value, name)
}
