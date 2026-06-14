package server

import (
	"context"
	"strings"
	"testing"
	"time"

	pb "github.com/baaaki/mydreamcampus/payment-service/proto"
	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestServer(t *testing.T) *PaymentServer {
	t.Helper()
	return NewPaymentServer(zap.NewNop())
}

func TestInitiatePayment_ReturnsSuccess(t *testing.T) {
	s := newTestServer(t)
	resp, err := s.InitiatePayment(context.Background(), &pb.InitiatePaymentRequest{
		ReferenceId: "ref-1",
		Amount:      150.5,
		Currency:    "TRY",
		StudentId:   "stud-1",
		Description: "Cafeteria reservation",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.True(t, resp.Success)
	assert.True(t, strings.HasPrefix(resp.PaymentId, "pay_"))
	assert.Contains(t, resp.PaymentUrl, resp.PaymentId)
	assert.Equal(t, 150.5, resp.Amount)
	assert.Equal(t, "TRY", resp.Currency)

	// expires_at must parse as RFC3339
	parsed, err := time.Parse(time.RFC3339, resp.ExpiresAt)
	require.NoError(t, err)
	assert.True(t, parsed.After(time.Now()), "expiry must be in the future")
}

func TestInitiatePayment_ExpiryUses30MinFromClockNow(t *testing.T) {
	frozen := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	clock.Set(frozen)
	defer clock.Reset()

	s := newTestServer(t)
	resp, err := s.InitiatePayment(context.Background(), &pb.InitiatePaymentRequest{
		ReferenceId: "ref", Amount: 1, Currency: "TRY", StudentId: "x",
	})
	require.NoError(t, err)

	parsed, err := time.Parse(time.RFC3339, resp.ExpiresAt)
	require.NoError(t, err)
	assert.Equal(t, frozen.Add(30*time.Minute), parsed)
}

func TestInitiatePayment_UniquePaymentIDs(t *testing.T) {
	s := newTestServer(t)
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		resp, err := s.InitiatePayment(context.Background(), &pb.InitiatePaymentRequest{
			ReferenceId: "x", Amount: 1, Currency: "TRY", StudentId: "x",
		})
		require.NoError(t, err)
		assert.False(t, seen[resp.PaymentId], "payment id collision at iter %d", i)
		seen[resp.PaymentId] = true
	}
}

func TestRequestRefund_ReturnsSuccess(t *testing.T) {
	s := newTestServer(t)
	resp, err := s.RequestRefund(context.Background(), &pb.RefundRequest{
		ReferenceId: "ref-1",
		Amount:      75.0,
		Currency:    "TRY",
		Reason:      "user cancelled",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.True(t, resp.Success)
	assert.True(t, strings.HasPrefix(resp.RefundId, "ref_"))
	assert.Equal(t, 75.0, resp.Amount)
	assert.Equal(t, "TRY", resp.Currency)
	assert.Equal(t, "completed", resp.Status)
	assert.NotEmpty(t, resp.Message)
}

func TestRequestRefund_AcceptsZeroAmount(t *testing.T) {
	// Mock service shouldn't enforce amount > 0; that's the caller's job.
	s := newTestServer(t)
	resp, err := s.RequestRefund(context.Background(), &pb.RefundRequest{
		ReferenceId: "x", Amount: 0, Currency: "TRY",
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}
