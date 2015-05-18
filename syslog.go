package logtap

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
)

// SyslogMessage type represents a parsed syslog message as defined by
// RFC5424. Caveat: it lacks "structured data" part because Heroku
// Logplex doesn't include it in HTTP requests it sends.
//
// See: http://tools.ietf.org/html/rfc5424#section-6
type SyslogMessage struct {
	Priority  string
	Version   string
	Timestamp time.Time
	Hostname  string
	Appname   string
	Procid    string
	Msgid     string
	Text      string
	// Heroku syslog data lacks STRUCTURED-DATA piece (which should be between Msgid and Text)
}

// ensureUtf8 produces a valid utf-8 encoded string. In case its input is
// invalid, all bad characters are replaced with \ufffd (aka "error"
// rune).
func ensureUtf8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	buf := bytes.Buffer{}
	start := 0
	for i := 0; i < len(s); {
		if s[i] < utf8.RuneSelf {
			i++
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteRune(utf8.RuneError)
			i++
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		buf.WriteString(s[start:])
	}
	return buf.String()
}

// messageRx matches stuff like this:
// <13>1 2014-01-09T04:06:38.793094+00:00 host app web.8 - Release v1822 created by foo@example.com
var syslogMessagePattern = regexp.MustCompile(`^<(.+?)>(.+?) (.+?) (.+?) (.+?) (.+?) (.+?) (.*)`)

// ErrSyslogPatternMismatch indicates that input doesn't match the
// syslog message pattern.
var ErrSyslogPatternMismatch = errors.New("syslog message pattern mismatch")

// ParseSyslogMessage parses a slice of bytes containing syslog
// message.
func ParseSyslogMessage(b []byte) (*SyslogMessage, error) {
	s := ensureUtf8(string(b))
	match := syslogMessagePattern.FindStringSubmatch(s)
	if match == nil {
		return nil, ErrSyslogPatternMismatch
	}
	t, err := time.Parse(time.RFC3339Nano, match[3])
	if err != nil {
		return nil, err
	}
	m := SyslogMessage{}
	m.Priority = match[1]
	m.Version = match[2]
	m.Timestamp = t.UTC()
	m.Hostname = match[4]
	m.Appname = match[5]
	m.Procid = match[6]
	m.Msgid = match[7]
	m.Text = match[8]
	return &m, nil
}

// tokenize follows bufio.SplitFunc protocol.
func tokenize(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, ' '); i >= 0 {
		size, err := strconv.ParseUint(string(data[:i]), 10, 16)
		if err == nil && len(data)-i-1 >= int(size) {
			return i + 1 + int(size), data[i+1 : i+1+int(size)], nil
		}
		return 0, nil, err
	}
	return 0, nil, nil
}

// ReadSyslogMessages returns a slice of scanned syslog messages from
// the specified reader using the syslog TCP protocol octet counting
// framing method. It appends messages to the specified slice,
// capacity permitting. It returns the updated slice. Basically, it
// operates on the slice exactly like built-in 'append' function. It
// also returns a potentially non-empty slice of errors that might
// have occurred during scanning.
//
// See the spec: http://tools.ietf.org/html/draft-gerhards-syslog-plain-tcp-12#section-3.4.1
func ReadSyslogMessages(results []*SyslogMessage, r io.Reader) ([]*SyslogMessage, []error) {
	var errors []error
	s := bufio.NewScanner(r)
	s.Split(tokenize)
	for s.Scan() {
		if m, err := ParseSyslogMessage(s.Bytes()); err == nil {
			results = append(results, m)
		} else {
			errors = append(errors, err)
		}
	}
	if err := s.Err(); err != nil {
		errors = append(errors, err)
	}
	return results, errors
}
