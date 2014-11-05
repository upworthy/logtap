// Package logtap provides components for Heroku Logplex HTTP drain
// processing.
package logtap

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

// ContextGetter represents the logic that obtains Logplex
// request-specific context that will be added to each received syslog
// message.
type ContextGetter interface {
	GetContext(r *http.Request) (interface{}, error)
}

// ContextFunc type is an adapter to allow the use of ordinary
// functions as ContextGetter-s.
type ContextFunc func(r *http.Request) (interface{}, error)

// GetContext calls f(r).
func (f ContextFunc) GetContext(r *http.Request) (interface{}, error) {
	return f(r)
}

// NilContext is for situations when request-specific context is not
// needed.
func NilContext(*http.Request) (interface{}, error) {
	return nil, nil
}

var errDrainTokenMissing = errors.New("request header 'Logplex-Drain-Token' is missing")

// GetDrainToken returns the drain token sent by Logplex in each
// request.
func GetDrainToken(r *http.Request) (interface{}, error) {
	if t := r.Header.Get("Logplex-Drain-Token"); t != "" {
		return t, nil
	}
	return nil, errDrainTokenMissing
}

var errAppNameMissing = errors.New("query string argument 'app' is missing")

// GetAppName returns the value of 'app' argument in the request query
// string.
func GetAppName(r *http.Request) (interface{}, error) {
	if app := r.URL.Query().Get("app"); app != "" {
		return app, nil
	}
	return nil, errAppNameMissing
}

// A Handler is a log tapping endpoint that processes syslog messages
// sent by Heroku Logplex.
//
// See: https://devcenter.heroku.com/articles/labs-https-drains
//
// Each successfully parsed syslog message is passed to the specified
// function F.
//
// Each syslog message will be enriched with "context" data derived
// from HTTP request. By default, Context field of each syslog message
// will be set to the value of 'app' query string argument. Context
// may be arbitrarily customized by setting ContextGetter field.
//
// Handler reports its operational state via Telemetry. Telemetry
// field may be set to customize how telemetry data is processed. Two
// canned Telemetry implementations are offered: DiscardTelemetry and
// LogTelemetry (the default one that uses the standard Go logging).
type Handler struct {
	Telemetry
	ContextGetter
	F func(*SyslogMessage)
}

// NewHandler creates a new instance of the log tapping endpoint that
// will invoke f for each received syslog message.
func NewHandler(f func(*SyslogMessage)) *Handler {
	h := Handler{
		LogTelemetry,
		ContextFunc(GetAppName),
		f,
	}
	return &h
}

// SetOptions customizes the Handler.
func (h *Handler) SetOptions(opts ...interface{}) *Handler {
	for _, o := range opts {
		switch o := o.(type) {
		case Telemetry:
			h.Telemetry = o
		case func(*http.Request) (interface{}, error):
			h.ContextGetter = ContextFunc(o)
		case ContextGetter:
			h.ContextGetter = o
		case func(*SyslogMessage):
			h.F = o
		}
	}
	return h
}

// ServeHTTP implements the log tapping endpoint logic.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ctx, err := h.ContextGetter.GetContext(r); err != nil {
		http.Error(w, err.Error(), http.StatusTeapot)
		h.Telemetry.Count(1, "context error")
	} else {
		var results []SyslogResult
		expectedCount, _ := strconv.Atoi(r.Header.Get("Logplex-Msg-Count"))
		if expectedCount > 0 && expectedCount <= 10 {
			// empirical evidence suggests that the upper bound of messages per Logplex request is 10.
			results = make([]SyslogResult, 0, expectedCount)
		}
		i := 0
		for _, result := range ReadSyslogMessages(results, r.Body) {
			if result.Err == nil {
				result.Message.Context = ctx
				h.F(result.Message)
				i++
				h.Telemetry.Value(time.Since(result.Message.Timestamp).Seconds(), "time lag")
			} else {
				log.Print(result.Err)
			}
		}
		h.Telemetry.Count(1, "request")
		if expectedCount > 0 {
			if i != expectedCount {
				log.Printf("Logplex-Msg-Count is %v, but %v messages have been read", expectedCount, i)
			}
			h.Telemetry.Value(expectedCount-i, "message count delta")
		}
	}
}
