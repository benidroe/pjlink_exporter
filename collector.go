package main

import (
	"context"
	"github.com/go-kit/kit/log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	ctx    context.Context
	target string
	pass   string
	//module *config.Module
	logger log.Logger
}

// Describe implements Prometheus.Collector.
func (c collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

// Collect implements Prometheus.Collector.
func (c collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	// pdus, err := ScrapeTarget(c.ctx, c.target, c.module, c.logger)
	/*if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("snmp_error", "Error scraping target", nil, nil), err)
		return
	}*/

	var pjSlice []prometheus.Metric // place to push collected metrics

	walkpjlink(c.target, c.pass, &pjSlice, c.logger)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_walk_duration_seconds", "Time PJLink walk took.", nil, nil),
		prometheus.GaugeValue,
		time.Since(start).Seconds())

	// iterate over results and push results to chan
	for i := range pjSlice {
		ch <- pjSlice[i]
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_scrape_duration_seconds", "Total PJLink time scrape took (walk and processing).", nil, nil),
		prometheus.GaugeValue,
		time.Since(start).Seconds())
}
