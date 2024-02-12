package async_txsub

import "github.com/prometheus/client_golang/prometheus"

type AsyncTxSubMetrics struct {
	// SubmissionDuration exposes timing metrics about the rate and latency of
	// submissions to stellar-core
	SubmissionDuration prometheus.Summary

	// OpenSubmissionsGauge tracks the count of "open" submissions (i.e.
	// submissions whose transactions haven't been confirmed successful or failed
	OpenSubmissionsGauge prometheus.Gauge

	// FailedSubmissionsCounter tracks the rate of failed transactions that have
	// been submitted to this process
	FailedSubmissionsCounter prometheus.Counter

	// SuccessfulSubmissionsCounter tracks the rate of successful transactions that
	// have been submitted to this process
	SuccessfulSubmissionsCounter prometheus.Counter

	// V0TransactionsCounter tracks the rate of v0 transaction envelopes that
	// have been submitted to this process
	V0TransactionsCounter prometheus.Counter

	// V1TransactionsCounter tracks the rate of v1 transaction envelopes that
	// have been submitted to this process
	V1TransactionsCounter prometheus.Counter

	// FeeBumpTransactionsCounter tracks the rate of fee bump transaction envelopes that
	// have been submitted to this process
	FeeBumpTransactionsCounter prometheus.Counter
}

func (metrics *AsyncTxSubMetrics) MustRegister(registry *prometheus.Registry) {
	registry.MustRegister(metrics.FailedSubmissionsCounter)
	registry.MustRegister(metrics.OpenSubmissionsGauge)
}
