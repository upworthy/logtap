package logtap

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"
	"unicode/utf8"
)

func TestEnsureUtf8(t *testing.T) {
	// f checks the ensureUtf8's invariant: no matter how terribly
	// non-utf8 the input is, the output will be a valid utf8 string.
	f := func(b []byte) bool {
		s := ensureUtf8(string(b))
		return utf8.ValidString(s)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestParseSyslogMessage(t *testing.T) {
	_, err := ParseSyslogMessage([]byte(
		"<45>1 2014-01-09T20:34:44.651004+00:00 host heroku api -" +
			" Add ZOMGZOMG config by foo@example.com"))
	if err != nil {
		t.Error(err)
	}
	_, err = ParseSyslogMessage([]byte(
		"<45>1 2OI4-01-09T20:34:44.651004+00:00 host heroku api -" +
			" Add ZOMGZOMG config by foo@example.com"))
	if err == nil {
		t.Error("Expected to fail when given malformed timestamp!")
	}
	f := func(b []byte) bool {
		// property: there's zero chance random bytes will parse as
		// valid syslog message.
		_, err := ParseSyslogMessage(b)
		return err == ErrSyslogPatternMismatch
	}
	if err = quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTokenize(t *testing.T) {
	advance, data, err := tokenize(nil, true)
	if advance != 0 || data != nil || err != nil {
		t.Error("Expecting 0, nil, nil when tokenize is at EOF and has no more data")
	}
	advance, data, err = tokenize([]byte("65536 YOLO..."), true)
	if advance != 0 || data != nil || err == nil {
		t.Error("Expecting 0, nil, err when tokenize encounters message length bigger than uint16")
	}
	f := func(expected []byte) bool {
		b := &bytes.Buffer{}
		b.WriteString(fmt.Sprintf("%v ", len(expected)))
		b.Write(expected)
		advance, data, err = tokenize(b.Bytes(), false)
		return advance == len(b.Bytes()) && reflect.DeepEqual(data, expected) && err == nil
	}
	if err = quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func testReadInvalidSyslogMessages(t *testing.T) {
	f := func(b []byte) bool {
		// property: totally random byte input will never produce messages.
		xs, _ := ReadSyslogMessages(nil, bytes.NewBuffer(b))
		if len(xs) != 0 {
			t.Errorf("Got %v unexpected syslog messages", len(xs))
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestReadSyslogMessages(t *testing.T) {
	utc, _ := time.LoadLocation("UTC")
	expected := []*SyslogMessage{
		&SyslogMessage{
			Priority:  "45",
			Version:   "1",
			Timestamp: time.Date(2014, 1, 9, 20, 34, 44, 651004000, utc),
			Hostname:  "host",
			Appname:   "heroku",
			Procid:    "api",
			Msgid:     "-",
			Text:      "Add ZOMGZOMG config by foo@example.com"},
		&SyslogMessage{
			Priority:  "45",
			Version:   "1",
			Timestamp: time.Date(2014, 1, 9, 20, 34, 44, 693891000, utc),
			Hostname:  "host",
			Appname:   "heroku",
			Procid:    "api",
			Msgid:     "-",
			Text:      "Release v1822 created by foo@example.com"},
	}
	actual, errors := ReadSyslogMessages(nil, strings.NewReader(
		`95 <45>1 2014-01-09T20:34:44.651004+00:00 host heroku api - Add ZOMGZOMG config by foo@example.com`+
			`97 <45>1 2014-01-09T20:34:44.693891+00:00 host heroku api - Release v1822 created by foo@example.com`+
			`zomg bogus`))
	if len(actual) != len(expected) {
		t.Errorf("Unexpected number of results: %v", len(actual))
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%#v != %#v", actual, expected)
	}
	if len(errors) != 1 {
		t.Errorf("Unexpected number of errors: %v", len(errors))
	}
}
