package brokerwithmetrics

import (
	"context"
	"runtime"
	brokerM "therealbroker/pkg/broker"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	methodCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "broker_service_method_count_total",
			Help: "Total number of RPC calls by method and status",
		},
		[]string{"method", "status"},
	)

	methodDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "broker_service_method_duration_seconds",
			Help:    "Duration of RPC calls",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	activeSubscribers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "broker_service_active_subscribers",
			Help: "Total number of active subscriptions",
		},
	)
	memoryUsage = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "app_memory_usage_bytes",
			Help: "Current memory usage in bytes.",
		},
		func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.Alloc)
		},
	)

	gcCycles = promauto.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "app_gc_cycles_total",
			Help: "Total number of completed GC cycles.",
		},
		func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.NumGC)
		},
	)
)

type BrokerWithMetrics struct {
	Service brokerM.Broker
	Log     *logrus.Logger
}

func (b *BrokerWithMetrics) Publish(ctx context.Context, subject string, msg brokerM.Message) (int, error) {
	start := time.Now()
	id, err := b.Service.Publish(ctx, subject, msg)
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
		b.Log.WithFields(logrus.Fields{
			"method": "Publish",
			"error":  err,
		}).Error("Publish failed")
	} else {
		b.Log.WithFields(logrus.Fields{
			"method": "Publish",
			"id":     id,
		}).Info("Message published")
	}

	methodCount.WithLabelValues("Publish", status).Inc()
	methodDuration.WithLabelValues("Publish").Observe(duration)

	return id, err
}

func (b *BrokerWithMetrics) Subscribe(ctx context.Context, subject string) (<-chan brokerM.Message, error) {
	activeSubscribers.Inc()
	ch, err := b.Service.Subscribe(ctx, subject)
	if err != nil {
		activeSubscribers.Dec()
		b.Log.WithFields(logrus.Fields{
			"method": "Subscribe",
			"error":  err,
		}).Error("Subscribe failed")
	} else {
		b.Log.WithFields(logrus.Fields{
			"method": "Subscribe",
		}).Info("Subscribed to subject")
	}
	return ch, err
}

func (b *BrokerWithMetrics) Fetch(ctx context.Context, subject string, id int) (brokerM.Message, error) {
	start := time.Now()
	msg, err := b.Service.Fetch(ctx, subject, id)
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
		b.Log.WithFields(logrus.Fields{
			"method": "Fetch",
			"id":     id,
			"error":  err,
		}).Error("Fetch failed")
	} else {
		b.Log.WithFields(logrus.Fields{
			"method": "Fetch",
			"id":     id,
		}).Info("Message fetched")
	}

	methodCount.WithLabelValues("Fetch", status).Inc()
	methodDuration.WithLabelValues("Fetch").Observe(duration)

	return msg, err
}

func (b *BrokerWithMetrics) Close() error {
	err := b.Service.Close()
	if err != nil {
		b.Log.WithFields(logrus.Fields{
			"method": "Close",
			"error":  err,
		}).Error("Service close failed")
	} else {
		b.Log.WithFields(logrus.Fields{
			"method": "Close",
		}).Info("Service closed")
	}
	return err
}
