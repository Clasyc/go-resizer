package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Counters struct {
	Resized map[string]int
}

func (a *Application) UpResized(label string) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()
	if _, ok := a.Counters.Resized[label]; !ok {
		a.Counters.Resized[label] = 0
	}
	a.Counters.Resized[label]++
}

type Exporter struct {
	ExporterMetrics
}

type ExporterMetrics struct {
	Resized *prometheus.Desc
}

func (em *ExporterMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- em.Resized
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
}

func (em *ExporterMetrics) initializeDescriptors() {
	em.Resized = prometheus.NewDesc(
		"resized_images_total",
		"Number of images resized",
		[]string{"label"},
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
