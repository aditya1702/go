package stellarcore

import (
	"context"

	"github.com/stretchr/testify/mock"

	proto "github.com/stellar/go/protocols/stellarcore"
)

type MockClientWithMetrics struct {
	mock.Mock
}

// SubmitTx mocks the SubmitTransaction method
func (m *MockClientWithMetrics) SubmitTx(ctx context.Context, rawTx string) (*proto.TXResponse, error) {
	args := m.Called(ctx, rawTx)
	return args.Get(0).(*proto.TXResponse), args.Error(1)
}
