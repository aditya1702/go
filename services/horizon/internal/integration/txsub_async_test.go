package integration

import (
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAsyncTxSub_SuccessfulSubmission(t *testing.T) {
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()
	masterAccount := itest.MasterAccount()

	txParams := txnbuild.TransactionParams{
		BaseFee:              txnbuild.MinBaseFee,
		SourceAccount:        masterAccount,
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: master.Address(),
				Amount:      "10",
				Asset:       txnbuild.NativeAsset{},
			},
		},
		Preconditions: txnbuild.Preconditions{
			TimeBounds:   txnbuild.NewInfiniteTimeout(),
			LedgerBounds: &txnbuild.LedgerBounds{MinLedger: 0, MaxLedger: 100},
		},
	}

	txResp, err := itest.AsyncSubmitTransaction(master, txParams)
	assert.NoError(t, err)
	assert.Equal(t, txResp, horizon.AsyncTransactionSubmissionResponse{
		ErrorResultXDR:      "",
		DiagnosticEventsXDR: "",
		TxStatus:            "PENDING",
		HttpStatus:          201,
		Hash:                "6cbb7f714bd08cea7c30cab7818a35c510cbbfc0a6aa06172a1e94146ecf0165",
	})
}

func TestAsyncTxSub_SubmissionError(t *testing.T) {
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()
	masterAccount := itest.MasterAccount()

	txParams := txnbuild.TransactionParams{
		BaseFee:              txnbuild.MinBaseFee,
		SourceAccount:        masterAccount,
		IncrementSequenceNum: false,
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: master.Address(),
				Amount:      "10",
				Asset:       txnbuild.NativeAsset{},
			},
		},
		Preconditions: txnbuild.Preconditions{
			TimeBounds:   txnbuild.NewInfiniteTimeout(),
			LedgerBounds: &txnbuild.LedgerBounds{MinLedger: 0, MaxLedger: 100},
		},
	}

	txResp, err := itest.AsyncSubmitTransaction(master, txParams)
	assert.NoError(t, err)
	assert.Equal(t, txResp, horizon.AsyncTransactionSubmissionResponse{
		ErrorResultXDR:      "AAAAAAAAAGT////7AAAAAA==",
		DiagnosticEventsXDR: "",
		TxStatus:            "ERROR",
		HttpStatus:          400,
		Hash:                "0684df00f20efd5876f1b8d17bc6d3a68d8b85c06bb41e448815ecaa6307a251",
	})
}

func TestAsyncTxSub_SubmissionTryAgainLater(t *testing.T) {
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()
	masterAccount := itest.MasterAccount()

	txParams := txnbuild.TransactionParams{
		BaseFee:              txnbuild.MinBaseFee,
		SourceAccount:        masterAccount,
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: master.Address(),
				Amount:      "10",
				Asset:       txnbuild.NativeAsset{},
			},
		},
		Preconditions: txnbuild.Preconditions{
			TimeBounds:   txnbuild.NewInfiniteTimeout(),
			LedgerBounds: &txnbuild.LedgerBounds{MinLedger: 0, MaxLedger: 100},
		},
	}

	txResp, err := itest.AsyncSubmitTransaction(master, txParams)
	assert.NoError(t, err)
	assert.Equal(t, txResp, horizon.AsyncTransactionSubmissionResponse{
		ErrorResultXDR:      "",
		DiagnosticEventsXDR: "",
		TxStatus:            "PENDING",
		HttpStatus:          201,
		Hash:                "6cbb7f714bd08cea7c30cab7818a35c510cbbfc0a6aa06172a1e94146ecf0165",
	})

	txResp, err = itest.AsyncSubmitTransaction(master, txParams)
	assert.NoError(t, err)
	assert.Equal(t, txResp, horizon.AsyncTransactionSubmissionResponse{
		ErrorResultXDR:      "",
		DiagnosticEventsXDR: "",
		TxStatus:            "TRY_AGAIN_LATER",
		HttpStatus:          503,
		Hash:                "d5eb72a4c1832b89965850fff0bd9bba4b6ca102e7c89099dcaba5e7d7d2e049",
	})
}
