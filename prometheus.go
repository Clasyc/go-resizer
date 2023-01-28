package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Counters struct {
	Resized  map[string]int
	Failures int
}

func (a *Application) UpResized(label string) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()
	if _, ok := a.Counters.Resized[label]; !ok {
		a.Counters.Resized[label] = 0
	}
	a.Counters.Resized[label]++
}

func (a *Application) UpFailures() {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()
	a.Counters.Failures++
}

type Exporter struct {
	ExporterMetrics
}

type ExporterMetrics struct {
	Resized  *prometheus.Desc
	Failures *prometheus.Desc
}

func (em *ExporterMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- em.Resized
	ch <- em.Failures
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for label, count := range app.Counters.Resized {
		ch <- prometheus.MustNewConstMetric(
			e.Resized,
			prometheus.CounterValue,
			float64(count),
			label,
		)
	}
	ch <- prometheus.MustNewConstMetric(e.Failures, prometheus.CounterValue, float64(app.Counters.Failures))
}

func (em *ExporterMetrics) initializeDescriptors() {
	em.Resized = prometheus.NewDesc(
		"resized_images_total",
		"Number of images resized",
		[]string{"label"},
		nil,
	)
	em.Failures = prometheus.NewDesc(
		"errors_total",
		"Number of times an error occurred",
		nil,
		nil,
	)
}

func NewExporter() *Exporter {
	em := ExporterMetrics{}
	em.initializeDescriptors()
	return &Exporter{
		ExporterMetrics: em,
	}
}
