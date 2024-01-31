package actions

import (
	"encoding/hex"
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/network"
	proto "github.com/stellar/go/protocols/stellarcore"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
	"mime"
	"net/http"
)

const (
	HttpStatusCodeForPending       = http.StatusCreated
	HttpStatusCodeForDuplicate     = http.StatusConflict
	HttpStatusCodeForTryAgainLater = http.StatusServiceUnavailable
	HttpStatusCodeForError         = http.StatusBadRequest
)

type AsyncSubmitTransactionHandler struct {
	NetworkPassphrase string
	DisableTxSub      bool
	CoreClient        stellarcore.ClientInterface
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

func (handler AsyncSubmitTransactionHandler) extractEnvelopeInfo(raw string, passphrase string) (envelopeInfo, error) {
	result := envelopeInfo{raw: raw}
	err := xdr.SafeUnmarshalBase64(raw, &result.parsed)
	if err != nil {
		return result, err
	}

	var hash [32]byte
	hash, err = network.HashTransactionInEnvelope(result.parsed, passphrase)
	if err != nil {
		return result, err
	}
	result.hash = hex.EncodeToString(hash[:])
	if result.parsed.IsFeeBump() {
		hash, err = network.HashTransaction(result.parsed.FeeBump.Tx.InnerTx.V1.Tx, passphrase)
		if err != nil {
			return result, err
		}
		result.innerHash = hex.EncodeToString(hash[:])
	}
	return result, nil
}

func (handler AsyncSubmitTransactionHandler) validateBodyType(r *http.Request) error {
	c := r.Header.Get("Content-Type")
	if c == "" {
		return nil
	}

	mt, _, err := mime.ParseMediaType(c)
	if err != nil {
		return errors.Wrap(err, "Could not determine mime type")
	}

	if mt != "application/x-www-form-urlencoded" && mt != "multipart/form-data" {
		return &hProblem.UnsupportedMediaType
	}
	return nil
}

func (handler AsyncSubmitTransactionHandler) GetResource(_ HeaderWriter, r *http.Request) (interface{}, error) {
	if err := handler.validateBodyType(r); err != nil {
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

	info, err := handler.extractEnvelopeInfo(raw, handler.NetworkPassphrase)
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

	resp, err := handler.CoreClient.SubmitTransaction(r.Context(), info.raw)
	if err != nil {
		return nil, &problem.P{
			Type:   "transaction_submission_failed",
			Title:  "Transaction Submission Failed",
			Status: http.StatusBadRequest,
			Detail: "Could not submit transaction to stellar-core. " +
				"The `extras.error` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-v2/transaction_submission_failed",
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
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-v2/transaction_submission_exception",
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
			HttpStatus:          HttpStatusCodeForError,
			Hash:                info.hash,
		}, nil
	case proto.TXStatusPending, proto.TXStatusDuplicate, proto.TXStatusTryAgainLater:
		var httpStatus int
		if resp.Status == proto.TXStatusPending {
			httpStatus = HttpStatusCodeForPending
		} else if resp.Status == proto.TXStatusDuplicate {
			httpStatus = HttpStatusCodeForDuplicate
		} else {
			httpStatus = HttpStatusCodeForTryAgainLater
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
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-v2/transaction_submission_invalid_status",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
				"error":        resp.Error,
			},
		}
	}

}
