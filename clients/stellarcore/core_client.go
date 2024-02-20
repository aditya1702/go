package stellarcore

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	proto "github.com/stellar/go/protocols/stellarcore"
	"time"
)

type CoreClientWithMetricsInterface interface {
	SubmitTransaction(ctx context.Context, envelope string) (resp *proto.TXResponse, err error)
}

type CoreClientWithMetrics struct {
	CoreClient ClientInterface

	AsyncTxSubMetrics struct {
		// AsyncSubmissionDuration exposes timing metrics about the rate and latency of
		// submissions to stellar-core
		AsyncSubmissionDuration *prometheus.SummaryVec

		// AsyncSubmissionsCounter tracks the rate of transactions that have
		// been submitted to this process
		AsyncSubmissionsCounter *prometheus.CounterVec

		// AsyncV0TransactionsCounter tracks the rate of v0 transaction envelopes that
		// have been submitted to this process
		AsyncV0TransactionsCounter *prometheus.CounterVec

		// AsyncV1TransactionsCounter tracks the rate of v1 transaction envelopes that
		// have been submitted to this process
		AsyncV1TransactionsCounter *prometheus.CounterVec

		// AsyncFeeBumpTransactionsCounter tracks the rate of fee bump transaction envelopes that
		// have been submitted to this process
		AsyncFeeBumpTransactionsCounter *prometheus.CounterVec
	}
}

func (c *CoreClientWithMetrics) SubmitTransaction(ctx context.Context, envelope string) (*proto.TXResponse, error) {
	startTime := time.Now()
	response, err := c.CoreClient.SubmitTransaction(ctx, envelope)
	duration := time.Since(startTime).Seconds()

	var label prometheus.Labels
	if err != nil {
		label = prometheus.Labels{"status": "request_error"}
	} else if response.IsException() {
		label = prometheus.Labels{"status": "exception"}
	} else {
		label = prometheus.Labels{"status": response.Status}
	}

	c.AsyncTxSubMetrics.AsyncSubmissionDuration.With(label).Observe(duration)
	return response, err
}

func NewCoreClientWithMetrics(client Client, registry *prometheus.Registry) *CoreClientWithMetrics {
	asyncSubmissionDuration := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "horizon",
		Subsystem:  "async_txsub",
		Name:       "submission_duration_seconds",
		Help:       "submission durations to Stellar-Core, sliding window = 10m",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"status"})
	asyncSubmissionsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "horizon",
		Subsystem: "async_txsub",
		Name:      "submissions_count",
		Help:      "number of submissions using the async txsub endpoint, sliding window = 10m",
	}, []string{"status"})
	asyncV0TransactionsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "horizon",
		Subsystem: "async_txsub",
		Name:      "v0_count",
		Help:      "number of v0 transaction envelopes submitted, sliding window = 10m",
	}, []string{"status"})
	asyncV1TransactionsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "horizon",
		Subsystem: "async_txsub",
		Name:      "v1_count",
		Help:      "number of v1 transaction envelopes submitted, sliding window = 10m",
	}, []string{"status"})
	asyncFeeBumpTransactionsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "horizon",
		Subsystem: "async_txsub",
		Name:      "feebump_count",
		Help:      "number of fee bump transaction envelopes submitted, sliding window = 10m",
	}, []string{"status"})

	registry.MustRegister(
		asyncSubmissionDuration,
		asyncSubmissionsCounter,
		asyncV0TransactionsCounter,
		asyncV1TransactionsCounter,
		asyncFeeBumpTransactionsCounter,
	)

	return &CoreClientWithMetrics{
		CoreClient: &client,
		AsyncTxSubMetrics: struct {
			AsyncSubmissionDuration         *prometheus.SummaryVec
			AsyncSubmissionsCounter         *prometheus.CounterVec
			AsyncV0TransactionsCounter      *prometheus.CounterVec
			AsyncV1TransactionsCounter      *prometheus.CounterVec
			AsyncFeeBumpTransactionsCounter *prometheus.CounterVec
		}{
			AsyncSubmissionDuration:         asyncSubmissionDuration,
			AsyncSubmissionsCounter:         asyncSubmissionsCounter,
			AsyncV0TransactionsCounter:      asyncV0TransactionsCounter,
			AsyncV1TransactionsCounter:      asyncV1TransactionsCounter,
			AsyncFeeBumpTransactionsCounter: asyncFeeBumpTransactionsCounter,
		},
	}
}
