package actions

import (
	"context"
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/network"
	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/services/horizon/internal/corestate"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const (
	TxXDR  = "AAAAAAGUcmKO5465JxTSLQOQljwk2SfqAJmZSG6JH6wtqpwhAAABLAAAAAAAAAABAAAAAAAAAAEAAAALaGVsbG8gd29ybGQAAAAAAwAAAAAAAAAAAAAAABbxCy3mLg3hiTqX4VUEEp60pFOrJNxYM1JtxXTwXhY2AAAAAAvrwgAAAAAAAAAAAQAAAAAW8Qst5i4N4Yk6l+FVBBKetKRTqyTcWDNSbcV08F4WNgAAAAAN4Lazj4x61AAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABLaqcIQAAAEBKwqWy3TaOxoGnfm9eUjfTRBvPf34dvDA0Nf+B8z4zBob90UXtuCqmQqwMCyH+okOI3c05br3khkH0yP4kCwcE"
	TxHash = "3389e9f0f1a65f19736cacf544c2e825313e8447f569233bb8db39aa607c8889"
)

func createRequest() *http.Request {
	form := url.Values{}
	form.Set("tx", TxXDR)

	request, _ := http.NewRequest(
		"POST",
		"http://localhost:8000/v2/transactions",
		strings.NewReader(form.Encode()),
	)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return request
}

func TestAsyncSubmitTransactionHandler_DisabledTxSub(t *testing.T) {
	handler := AsyncSubmitTransactionHandler{
		DisableTxSub: true,
	}

	request := createRequest()
	w := httptest.NewRecorder()

	_, err := handler.GetResource(w, request)
	assert.NotNil(t, err)
	assert.IsType(t, &problem.P{}, err)
	p := err.(*problem.P)
	assert.Equal(t, "transaction_submission_disabled", p.Type)
	assert.Equal(t, http.StatusMethodNotAllowed, p.Status)
}

func TestAsyncSubmitTransactionHandler_MalformedTransaction(t *testing.T) {
	handler := AsyncSubmitTransactionHandler{}

	request := createRequest()
	w := httptest.NewRecorder()

	_, err := handler.GetResource(w, request)
	assert.NotNil(t, err)
	assert.IsType(t, &problem.P{}, err)
	p := err.(*problem.P)
	assert.Equal(t, "transaction_malformed", p.Type)
	assert.Equal(t, http.StatusBadRequest, p.Status)
}

func TestAsyncSubmitTransactionHandler_CoreNotSynced(t *testing.T) {
	coreStateGetter := new(coreStateGetterMock)
	coreStateGetter.On("GetCoreState").Return(corestate.State{Synced: false})
	handler := AsyncSubmitTransactionHandler{
		CoreStateGetter:   coreStateGetter,
		NetworkPassphrase: network.PublicNetworkPassphrase,
	}

	request := createRequest()
	w := httptest.NewRecorder()

	_, err := handler.GetResource(w, request)
	assert.NotNil(t, err)
	assert.IsType(t, problem.P{}, err)
	p := err.(problem.P)
	assert.Equal(t, "stale_history", p.Type)
	assert.Equal(t, http.StatusServiceUnavailable, p.Status)
}

func TestAsyncSubmitTransactionHandler_TransactionSubmissionFailed(t *testing.T) {
	coreStateGetter := new(coreStateGetterMock)
	coreStateGetter.On("GetCoreState").Return(corestate.State{Synced: true})

	mockCoreClient := &stellarcore.MockClient{}
	mockCoreClient.On("SubmitTransaction", context.Background(), TxXDR).Return(&proto.TXResponse{}, errors.Errorf("submission error"))

	handler := AsyncSubmitTransactionHandler{
		CoreStateGetter:   coreStateGetter,
		NetworkPassphrase: network.PublicNetworkPassphrase,
		CoreClient:        mockCoreClient,
	}

	request := createRequest()
	w := httptest.NewRecorder()

	_, err := handler.GetResource(w, request)
	assert.NotNil(t, err)
	assert.IsType(t, &problem.P{}, err)
	p := err.(*problem.P)
	assert.Equal(t, "transaction_submission_failed", p.Type)
	assert.Equal(t, http.StatusBadRequest, p.Status)
}

func TestAsyncSubmitTransactionHandler_TransactionSubmissionException(t *testing.T) {
	coreStateGetter := new(coreStateGetterMock)
	coreStateGetter.On("GetCoreState").Return(corestate.State{Synced: true})

	mockCoreClient := &stellarcore.MockClient{}
	mockCoreClient.On("SubmitTransaction", context.Background(), TxXDR).Return(&proto.TXResponse{
		Exception: "some-exception",
	}, nil)

	handler := AsyncSubmitTransactionHandler{
		CoreStateGetter:   coreStateGetter,
		NetworkPassphrase: network.PublicNetworkPassphrase,
		CoreClient:        mockCoreClient,
	}

	request := createRequest()
	w := httptest.NewRecorder()

	_, err := handler.GetResource(w, request)
	assert.NotNil(t, err)
	assert.IsType(t, &problem.P{}, err)
	p := err.(*problem.P)
	assert.Equal(t, "transaction_submission_exception", p.Type)
	assert.Equal(t, http.StatusBadRequest, p.Status)
}

func TestAsyncSubmitTransactionHandler_TransactionStatusResponse(t *testing.T) {
	coreStateGetter := new(coreStateGetterMock)
	coreStateGetter.On("GetCoreState").Return(corestate.State{Synced: true})

	successCases := []struct {
		mockCoreResponse *proto.TXResponse
		expectedResponse TransactionSubmissionResponse
	}{
		{
			mockCoreResponse: &proto.TXResponse{
				Exception:        "",
				Error:            "test-error",
				Status:           proto.TXStatusError,
				DiagnosticEvents: "test-diagnostic-events",
			},
			expectedResponse: TransactionSubmissionResponse{
				ErrorResultXDR:      "test-error",
				DiagnosticEventsXDR: "test-diagnostic-events",
				TxStatus:            proto.TXStatusError,
				HttpStatus:          HttpStatusCodeForError,
				Hash:                TxHash,
			},
		},
		{
			mockCoreResponse: &proto.TXResponse{
				Status: proto.TXStatusPending,
			},
			expectedResponse: TransactionSubmissionResponse{
				TxStatus:   proto.TXStatusPending,
				HttpStatus: HttpStatusCodeForPending,
				Hash:       TxHash,
			},
		},
		{
			mockCoreResponse: &proto.TXResponse{
				Status: proto.TXStatusDuplicate,
			},
			expectedResponse: TransactionSubmissionResponse{
				TxStatus:   proto.TXStatusDuplicate,
				HttpStatus: HttpStatusCodeForDuplicate,
				Hash:       TxHash,
			},
		},
		{
			mockCoreResponse: &proto.TXResponse{
				Status: proto.TXStatusTryAgainLater,
			},
			expectedResponse: TransactionSubmissionResponse{
				TxStatus:   proto.TXStatusTryAgainLater,
				HttpStatus: HttpStatusCodeForTryAgainLater,
				Hash:       TxHash,
			},
		},
	}

	for _, testCase := range successCases {
		mockCoreClient := &stellarcore.MockClient{}
		mockCoreClient.On("SubmitTransaction", context.Background(), TxXDR).Return(testCase.mockCoreResponse, nil)

		handler := AsyncSubmitTransactionHandler{
			NetworkPassphrase: network.PublicNetworkPassphrase,
			CoreClient:        mockCoreClient,
			CoreStateGetter:   coreStateGetter,
		}

		request := createRequest()
		w := httptest.NewRecorder()

		resp, err := handler.GetResource(w, request)
		assert.NoError(t, err)
		assert.Equal(t, resp, testCase.expectedResponse)
	}
}
