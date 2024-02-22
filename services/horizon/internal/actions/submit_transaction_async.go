package actions

import (
	async_txsub "github.com/stellar/go/clients/stellarcore"
	proto "github.com/stellar/go/protocols/stellarcore"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/support/render/problem"
	"net/http"
)

const (
	HTTPStatusCodeForPending       = http.StatusCreated
	HTTPStatusCodeForDuplicate     = http.StatusConflict
	HTTPStatusCodeForTryAgainLater = http.StatusServiceUnavailable
	HTTPStatusCodeForError         = http.StatusBadRequest
)

type AsyncSubmitTransactionHandler struct {
	NetworkPassphrase string
	DisableTxSub      bool
	ClientWithMetrics async_txsub.ClientWithMetricsInterface
	CoreStateGetter
}

// TransactionSubmissionResponse represents the response returned by Horizon
// when using the transaction-submission-v2 endpoint.
type TransactionSubmissionResponse struct {
	// ErrorResultXDR is present only if Status is equal to proto.TXStatusError.
	// ErrorResultXDR is a TransactionResult xdr string which contains details on why
	// the transaction could not be accepted by stellar-core.
	ErrorResultXDR string `json:"errorResultXdr,omitempty"`
	// DiagnosticEventsXDR is present only if Status is equal to proto.TXStatusError.
	// DiagnosticEventsXDR is a base64-encoded slice of xdr.DiagnosticEvent
	DiagnosticEventsXDR string `json:"diagnosticEventsXdr,omitempty"`
	// TxStatus represents the status of the transaction submission returned by stellar-core.
	// It can be one of: proto.TXStatusPending, proto.TXStatusDuplicate,
	// proto.TXStatusTryAgainLater, or proto.TXStatusError.
	TxStatus string `json:"tx_status"`
	// HttpStatus represents the corresponding http status code.
	HttpStatus int `json:"status"`
	// Hash is a hash of the transaction which can be used to look up whether
	// the transaction was included in the ledger.
	Hash string `json:"hash"`
}

func (handler AsyncSubmitTransactionHandler) GetResource(_ HeaderWriter, r *http.Request) (interface{}, error) {
	if err := validateBodyType(r); err != nil {
		return nil, err
	}

	raw, err := getString(r, "tx")
	if err != nil {
		return nil, err
	}

	if handler.DisableTxSub {
		return nil, &problem.P{
			Type:   "transaction_submission_disabled",
			Title:  "Transaction Submission Disabled",
			Status: http.StatusMethodNotAllowed,
			Detail: "Transaction submission has been disabled for Horizon. " +
				"To enable it again, remove env variable DISABLE_TX_SUB.",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
			},
		}
	}

	info, err := extractEnvelopeInfo(raw, handler.NetworkPassphrase)
	if err != nil {
		return nil, &problem.P{
			Type:   "transaction_malformed",
			Title:  "Transaction Malformed",
			Status: http.StatusBadRequest,
			Detail: "Horizon could not decode the transaction envelope in this " +
				"request. A transaction should be an XDR TransactionEnvelope struct " +
				"encoded using base64.  The envelope read from this request is " +
				"echoed in the `extras.envelope_xdr` field of this response for your " +
				"convenience.",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
			},
		}
	}

	coreState := handler.GetCoreState()
	if !coreState.Synced {
		return nil, hProblem.StaleHistory
	}

	resp, err := handler.ClientWithMetrics.SubmitTransaction(r.Context(), info.raw, info.parsed)
	if err != nil {
		return nil, &problem.P{
			Type:   "transaction_submission_failed",
			Title:  "Transaction Submission Failed",
			Status: http.StatusBadRequest,
			Detail: "Could not submit transaction to stellar-core. " +
				"The `extras.error` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-async/transaction_submission_failed",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
				"error":        err,
			},
		}
	}

	if resp.IsException() {
		return nil, &problem.P{
			Type:   "transaction_submission_exception",
			Title:  "Transaction Submission Exception",
			Status: http.StatusBadRequest,
			Detail: "Received exception from stellar-core." +
				"The `extras.error` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-async/transaction_submission_exception",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
				"error":        resp.Exception,
			},
		}
	}

	switch resp.Status {
	case proto.TXStatusError:
		return TransactionSubmissionResponse{
			ErrorResultXDR:      resp.Error,
			DiagnosticEventsXDR: resp.DiagnosticEvents,
			TxStatus:            resp.Status,
			HttpStatus:          HTTPStatusCodeForError,
			Hash:                info.hash,
		}, nil
	case proto.TXStatusPending, proto.TXStatusDuplicate, proto.TXStatusTryAgainLater:
		var httpStatus int
		if resp.Status == proto.TXStatusPending {
			httpStatus = HTTPStatusCodeForPending
		} else if resp.Status == proto.TXStatusDuplicate {
			httpStatus = HTTPStatusCodeForDuplicate
		} else {
			httpStatus = HTTPStatusCodeForTryAgainLater
		}

		return TransactionSubmissionResponse{
			TxStatus:   resp.Status,
			HttpStatus: httpStatus,
			Hash:       info.hash,
		}, nil
	default:
		return nil, &problem.P{
			Type:   "transaction_submission_invalid_status",
			Title:  "Transaction Submission Invalid Status",
			Status: http.StatusBadRequest,
			Detail: "Received invalid status from stellar-core." +
				"The `extras.error` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-async/transaction_submission_invalid_status",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
				"error":        resp.Error,
			},
		}
	}

}
