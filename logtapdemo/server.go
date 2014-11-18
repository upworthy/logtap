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
	"time"
)

func quantify(name string, window time.Duration, c <-chan float64) {
	q := quantile.NewTargeted(0.50, 0.99)
	tick := time.NewTicker(window)
	for {
		select {
		case <-tick.C:
			n := q.Count()
			log.Printf("%16s: p50=%-9.2f p99=%-9.2f n=%v", name, q.Query(0.50), q.Query(0.99), n)
			q.Reset()
		case v := <-c:
			q.Insert(v)
		}
	}
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

	rps, requests := newCounter(time.Second)

	inputs := map[string]chan float64{
		"service": make(chan float64, 10000),
		"connect": make(chan float64, 10000),
	}

	f := func(m *logtap.SyslogMessage) {
		if m.Appname == "heroku" && m.Procid == "router" {
			requests <- 1
			for _, match := range timingsPattern.FindAllStringSubmatch(m.Text, -1) {
				if c, ok := inputs[match[1]]; ok {
					if v, err := strconv.ParseFloat(match[2], 64); err == nil {
						c <- v
					}
				}
			}
		}
	}

	go quantify("service", 10*time.Second, inputs["service"])
	go quantify("connect", 10*time.Second, inputs["connect"])
	go quantify("rps", 10*time.Second, rps)

	go http.Handle("/", logtap.NewHandler(f).SetOptions(logtap.NilContext, logtap.DiscardTelemetry))
	secureheader.DefaultConfig.PermitClearLoopback = permitClearLoopback
	log.Fatal(http.ListenAndServe(net.JoinHostPort(bindAddr, os.Getenv("PORT")), secureheader.DefaultConfig))
}
