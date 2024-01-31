package stellarcore

import (
	"context"
	"net/http"
	"net/url"

	"github.com/stellar/go/support/http/httptest"

	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mockable stellar-core client.
type MockClient struct {
	mock.Mock
}

// Upgrade mocks the Upgrade method
func (m *MockClient) Upgrade(ctx context.Context, version int) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

// GetLedgerEntry mocks the GetLedgerEntry method
func (m *MockClient) GetLedgerEntry(ctx context.Context, ledgerKey xdr.LedgerKey) (proto.GetLedgerEntryResponse, error) {
	args := m.Called(ctx, ledgerKey)
	return args.Get(0).(proto.GetLedgerEntryResponse), args.Error(1)
}

// Info mocks the Info method
func (m *MockClient) Info(ctx context.Context) (*proto.InfoResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*proto.InfoResponse), args.Error(1)
}

// SetCursor mocks the SetCursor method
func (m *MockClient) SetCursor(ctx context.Context, id string, cursor int32) error {
	args := m.Called(ctx, id, cursor)
	return args.Error(0)
}

// SubmitTransaction mocks the SubmitTransaction method
func (m *MockClient) SubmitTransaction(ctx context.Context, envelope string) (*proto.TXResponse, error) {
	args := m.Called(ctx, envelope)
	return args.Get(0).(*proto.TXResponse), args.Error(1)
}

// WaitForNetworkSync mocks the WaitForNetworkSync method
func (m *MockClient) WaitForNetworkSync(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// ManualClose mocks the ManualClose method
func (m *MockClient) ManualClose(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Mock for http() method
func (m *MockClient) http() HTTP {
	return httptest.NewClient()
}

// Mock for simpleGet() method
func (m *MockClient) simpleGet(ctx context.Context, newPath string, query url.Values) (*http.Request, error) {
	args := m.Called(ctx, newPath, query)
	return args.Get(0).(*http.Request), args.Error(1)
}
