package stellarcore

import (
	"context"
	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stretchr/testify/mock"
)

type MockCoreClientWithMetrics struct {
	mock.Mock
}

// SubmitTransaction mocks the SubmitTransaction method
func (m *MockCoreClientWithMetrics) SubmitTransaction(ctx context.Context, envelope string) (*proto.TXResponse, error) {
	args := m.Called(ctx, envelope)
	return args.Get(0).(*proto.TXResponse), args.Error(1)
}
