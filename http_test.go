package logtap

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestContextGetters(t *testing.T) {
	r, _ := http.NewRequest("POST", "https://examlpe.com/", nil)
	if _, err := GetAppName(r); err != errAppNameMissing {
		t.Error(err)
	}
	r, _ = http.NewRequest("POST", "https://examlpe.com/?app=qaz", nil)
	if app, err := GetAppName(r); app != "qaz" || err != nil {
		t.Error("GetAppName failed")
	}
	r, _ = http.NewRequest("POST", "https://examlpe.com/", nil)
	if _, err := GetDrainToken(r); err != errDrainTokenMissing {
		t.Error(err)
	}
	r, _ = http.NewRequest("POST", "https://examlpe.com/", nil)
	r.Header.Set("Logplex-Drain-Token", "123")
	if tok, err := GetDrainToken(r); tok != "123" || err != nil {
		t.Error("GetDrainToken failed")
	}
}

func TestHandlerServeHTTP(t *testing.T) {
	utc, _ := time.LoadLocation("UTC")
	d := strings.NewReader(
		"97 <45>1 2014-01-09T20:34:44.693891+00:00 host heroku api - Release v1822 created by foo@example.com" +
			"97 <45>1*2014-01-09T20:34:44.693891+00:00*host*heroku*api*-*Bogus entirely on purpose yes preciousss" +
			"23 BAD FRAMING...")
	r, _ := http.NewRequest("POST", "https://logtap.example.org/", d)
	r.Header.Set("Logplex-Msg-Count", "3")
	w := httptest.NewRecorder()
	var actual *SyslogMessage
	f := func(m *SyslogMessage) { actual = m }
	h := NewHandler(f)
	h.Telemetry = DiscardTelemetry
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal("HTTP status != 200")
	}
	expected := &SyslogMessage{
		Priority:  "45",
		Version:   "1",
		Timestamp: time.Date(2014, 1, 9, 20, 34, 44, 693891000, utc),
		Hostname:  "host",
		Appname:   "heroku",
		Procid:    "api",
		Msgid:     "-",
		Text:      "Release v1822 created by foo@example.com",
		Context:   nil,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%#v != %#v", actual, expected)
	}
}

func TestHandlerServeHTTPFailsWithoutContext(t *testing.T) {
	d := strings.NewReader("97 <45>1 2014-01-09T20:34:44.693891+00:00 host heroku api - Release v1822 created by foo@example.com")
	r, _ := http.NewRequest("POST", "https://logtap.example.org/", d)
	w := httptest.NewRecorder()
	f := func(*SyslogMessage) {}
	h := NewHandler(f)
	h.ContextGetter = ContextFunc(GetAppName)
	h.ServeHTTP(w, r)
	if w.Code != http.StatusTeapot {
		t.Fatal("HTTP status != StatusTeapot")
	}
	if w.Body.String() != "query string argument 'app' is missing\n" {
		t.Error("Unexpected body contents:", w.Body.String())
	}
}
