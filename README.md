# Logtap: Heroku syslog drain processing library

[Learn more][drains] about Heroku Logplex HTTP drains.

[![GoDoc](https://godoc.org/github.com/upworthy/logtap?status.png)](https://godoc.org/github.com/upworthy/logtap)

[drains]: https://devcenter.heroku.com/articles/labs-https-drains

## Demo

[![Deploy](https://www.herokucdn.com/deploy/button.png)](https://heroku.com/deploy)

Once you've deployed Logtap demo, you will need to drain your web
app's logs to Logtap. For example, if Logtap app got deployed as
`serene-wookie-1234` and your web app is `acme-www-prod`, you'll need
to do this:

    $ hk drain-add -a acme-www-prod https://serene-wookie-1234.herokuapp.com

You can now see Logtap demo producing some stats based on log messages
received from `acme-www-prod`:

    $ hk log -a serene-wookie-1234 -s app
    ...
    2014-11-14T06:26:43 app[web.1]: load_avg_1m: p50=0.56      p99=2.41      n=587
    2014-11-14T06:26:43 app[web.1]:     service: p50=23.00     p99=334.00    n=492
    2014-11-14T06:26:43 app[web.1]:     connect: p50=1.00      p99=6.00      n=492
    2014-11-14T06:26:43 app[web.1]:         rps: p50=51.00     p99=64.00     n=60
    2014-11-14T06:26:53 app[web.1]: load_avg_1m: p50=0.56      p99=2.41      n=589
    2014-11-14T06:26:53 app[web.1]:     service: p50=24.00     p99=317.00    n=474
    2014-11-14T06:26:53 app[web.1]:     connect: p50=1.00      p99=5.00      n=474
    2014-11-14T06:26:53 app[web.1]:         rps: p50=51.00     p99=64.00     n=70
    ...

To clean up after you're done with this demo, get a list of log drains for `acme-www-prod`:

    $ hk drains -a acme-www-prod
    2ba1aa24-eaaf-48b9-a629-208672bc920e  https://serene-wookie-1234.herokuapp.com
    ...

And remove the drain using its ID:

    $ hk drain-remove 2ba1aa24-eaaf-48b9-a629-208672bc920e
