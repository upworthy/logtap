package telemetry

import (
	"bufio"
	"bytes"
	"log"
	"reflect"
	"testing"
)

func TestLogMetrics(t *testing.T) {
	buf := &bytes.Buffer{}
	log.SetFlags(0)
	log.SetOutput(buf)
	LogMetrics.Value(3.14, "Pi")
	LogMetrics.Value(-17, "foo")
	LogMetrics.Count(42, "the answer")
	LogMetrics.Count(1, "love")
	LogMetrics.Count(1, "life")
	LogMetrics.Value("not a good value!", "BAD")
	s := bufio.NewScanner(buf)
	var actual []string
	for s.Scan() {
		actual = append(actual, s.Text())
	}
	expected := []string{
		`APP_METRIC {"stat": "Pi", "value": 3.14}`,
		`APP_METRIC {"stat": "foo", "value": -17}`,
		`APP_METRIC {"stat": "the answer", "count": 42}`,
		`APP_METRIC {"stat": "love", "count": 1}`,
		`APP_METRIC {"stat": "life", "count": 1}`,
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%#v != %#v", actual, expected)
	}
}
