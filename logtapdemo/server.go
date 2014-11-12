package main

import (
	"github.com/bmizerany/perks/quantile"
	"github.com/kr/secureheader"
	"github.com/upworthy/logtap"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func quantify(name string, window time.Duration, maxCount int, c <-chan float64) {
	q := quantile.NewTargeted(0.50, 0.99)
	tick := time.NewTicker(window)
	for {
		select {
		case <-tick.C:
			n := q.Count()
			log.Printf("%16s: p50=%-9.2f p99=%-9.2f n=%v", name, q.Query(0.50), q.Query(0.99), n)
			if n > maxCount {
				q.Reset()
			}
		case v := <-c:
			q.Insert(v)
		}
	}
}

func sendAsFloat(s string, c chan<- float64) {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		c <- v
	}
}

func findAll(p *regexp.Regexp, s string) <-chan []string {
	c := make(chan []string)
	go func() {
		defer close(c)
		for _, m := range p.FindAllStringSubmatch(s, -1) {
			c <- m
		}
	}()
	return c
}

func newCounter(window time.Duration) (<-chan float64, chan<- int) {
	counts := make(chan float64)
	incs := make(chan int, 10000)
	go func() {
		var n float64
		tick := time.NewTicker(window)
		for {
			select {
			case <-tick.C:
				select {
				case counts <- n:
				}
				n = 0
			case x := <-incs:
				n += float64(x)
			}
		}
	}()
	return counts, incs
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	timingsPattern := regexp.MustCompile(`(\S+)=(\d+)ms`)
	loadsPattern := regexp.MustCompile(`sample#(load_avg_\d+m)=(\S+)`)

	rps, requests := newCounter(time.Second)

	inputs := map[string]chan float64{
		"load_avg_1m": make(chan float64),
		"service":     make(chan float64, 10000),
		"connect":     make(chan float64, 10000),
	}

	f := func(m *logtap.SyslogMessage) {
		if m.Appname == "heroku" && m.Procid == "router" {
			requests <- 1
			for m := range findAll(timingsPattern, m.Text) {
				if c, ok := inputs[m[1]]; ok {
					sendAsFloat(m[2], c)
				}
			}
		} else if m.Appname == "heroku" && strings.HasPrefix(m.Procid, "web.") {
			for m := range findAll(loadsPattern, m.Text) {
				if c, ok := inputs[m[1]]; ok {
					sendAsFloat(m[2], c)
				}
			}
		}
	}

	go quantify("load_avg_1m", 10*time.Second, 1000, inputs["load_avg_1m"])
	go quantify("service", 10*time.Second, 0, inputs["service"])
	go quantify("connect", 10*time.Second, 0, inputs["connect"])
	go quantify("rps", 10*time.Second, 600, rps)

	go http.Handle("/", logtap.NewHandler(f).SetOptions(logtap.NilContext, logtap.DiscardTelemetry))
	secureheader.DefaultConfig.PermitClearLoopback = permitClearLoopback
	log.Fatal(http.ListenAndServe(net.JoinHostPort(bindAddr, os.Getenv("PORT")), secureheader.DefaultConfig))
}
