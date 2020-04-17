package main

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/version"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":2112").String()
	configFile    = kingpin.Flag("config.file", "Path to configuration file.").Default("pjlink.yml").String()

	// Metrics about the SNMP exporter itself.
	pjlinkDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "snmp_collection_duration_seconds",
			Help: "Duration of collections by the SNMP exporter",
		},
		[]string{"module"},
	)
	pjlinkRequestErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "snmp_request_errors_total",
			Help: "Errors in requests to the SNMP exporter",
		},
	)

	config = Config{}

	reloadCh chan chan error
)

func init() {
	prometheus.MustRegister(pjlinkDuration)
	prometheus.MustRegister(pjlinkRequestErrors)
	prometheus.MustRegister(version.NewCollector("pjlink_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request, logger log.Logger) {
	query := r.URL.Query()

	target := query.Get("target")
	if len(query["target"]) != 1 || target == "" {
		http.Error(w, "'target' parameter must be specified once", 400)

		return
	}

	logger = log.With(logger, "target", target)
	level.Debug(logger).Log("msg", "Starting scrape")

	start := time.Now()
	registry := prometheus.NewRegistry()
	collector := collector{ctx: r.Context(), target: target, pass: config.getDevicePassword(target), logger: logger}
	registry.MustRegister(collector)
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	duration := time.Since(start).Seconds()
	pjlinkDuration.WithLabelValues("PJLink").Observe(duration)
	level.Debug(logger).Log("msg", "Finished scrape", "duration_seconds", duration)
}

func main() {

	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	level.Info(logger).Log("msg", "Starting pjlink_exporter...")

	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	if err := config.readConfig(*configFile); err != nil {
		level.Error(logger).Log("msg", "Error reloading config", "err", err)
	} else {
		level.Info(logger).Log("msg", "Loaded config file "+*configFile)
	}

	hup := make(chan os.Signal, 1)
	reloadCh = make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-hup:
				if err := config.readConfig(*configFile); err != nil {
					level.Error(logger).Log("msg", "Error reloading config", "err", err)
				} else {
					level.Info(logger).Log("msg", "Loaded config file")
				}
			case rc := <-reloadCh:
				if err := config.readConfig(*configFile); err != nil {
					level.Error(logger).Log("msg", "Error reloading config", "err", err)
					rc <- err
				} else {
					level.Info(logger).Log("msg", "Loaded config file")
					rc <- nil
				}
			}
		}
	}()

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/pjlink", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, logger)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head>
            <title>PJLink Exporter</title>
            <style>
            label{
            display:inline-block;
            width:75px;
            }
            form label {
            margin: 10px;
            }
            form input {
            margin: 10px;
            }
            </style>
            </head>
            <body>
            <h1>PJLink Exporter</h1>
            <form action="/pjlink">
            <label>Target:</label> <input type="text" name="target" placeholder="X.X.X.X" value="1.2.3.4"><br>
            <input type="submit" value="Submit">
            </form>
						<p><a href="/config">Config</a></p>
            </body>
            </html>`))
	})

	http.ListenAndServe(*listenAddress, nil)
}
