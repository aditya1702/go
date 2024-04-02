package stellarcore

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/xdr"
)

type ClientWithMetrics interface {
	SubmitTransaction(ctx context.Context, rawTx string, envelope xdr.TransactionEnvelope) (resp *proto.TXResponse, err error)
}

type clientWithMetrics struct {
	CoreClient Client

	TxSubMetrics struct {
		// SubmissionDuration exposes timing metrics about the rate and latency of
		// submissions to stellar-core
		SubmissionDuration *prometheus.SummaryVec

		// V0TransactionsCounter tracks the rate of v0 transaction envelopes that
		// have been submitted to this process
		V0TransactionsCounter *prometheus.CounterVec

		// V1TransactionsCounter tracks the rate of v1 transaction envelopes that
		// have been submitted to this process
		V1TransactionsCounter *prometheus.CounterVec

		// FeeBumpTransactionsCounter tracks the rate of fee bump transaction envelopes that
		// have been submitted to this process
		FeeBumpTransactionsCounter *prometheus.CounterVec
	}
}

func (c *clientWithMetrics) SubmitTransaction(ctx context.Context, rawTx string, envelope xdr.TransactionEnvelope) (*proto.TXResponse, error) {
	startTime := time.Now()
	response, err := c.CoreClient.SubmitTransaction(ctx, rawTx)
	c.updateTxSubMetrics(time.Since(startTime).Seconds(), envelope, response, err)

	return response, err
}

func (c *clientWithMetrics) updateTxSubMetrics(duration float64, envelope xdr.TransactionEnvelope, response *proto.TXResponse, err error) {
	label := prometheus.Labels{}
	if err != nil {
		label["status"] = "request_error"
	} else if response.IsException() {
		label["status"] = "exception"
	} else {
		label["status"] = response.Status
	}

	switch envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTxV0:
		label["envelope_type"] = "v0"
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		label["envelope_type"] = "v1"
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		label["envelope_type"] = "fee-bump"
	}

	c.TxSubMetrics.SubmissionDuration.With(label).Observe(duration)
}

func NewClientWithMetrics(client Client, registry *prometheus.Registry, prometheusSubsystem string) ClientWithMetrics {
	submissionDuration := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "horizon",
		Subsystem:  prometheusSubsystem,
		Name:       "submission_duration_seconds",
		Help:       "submission durations to Stellar-Core, sliding window = 10m",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"status", "envelope_type"})

	registry.MustRegister(
		submissionDuration,
	)

	return &clientWithMetrics{
		CoreClient: client,
		TxSubMetrics: struct {
			SubmissionDuration         *prometheus.SummaryVec
			V0TransactionsCounter      *prometheus.CounterVec
			V1TransactionsCounter      *prometheus.CounterVec
			FeeBumpTransactionsCounter *prometheus.CounterVec
		}{
			SubmissionDuration: submissionDuration,
		},
	}
}
