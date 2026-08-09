package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/meal/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPaymentTestClient(t *testing.T, handler http.HandlerFunc) *HTTPPaymentClient {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewHTTPPaymentClient(client.NewBase(client.Config{
		Target:  "payment-service",
		BaseURL: srv.URL,
		Secret:  "test-secret",
	}))
}

func TestHTTPPaymentClient_InitiatePayment_ReturnsPayment(t *testing.T) {
	var got dto.InitiatePaymentRequest
	var gotPath, gotSecret string
	c := newPaymentTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotSecret = r.URL.Path, r.Header.Get("X-Internal-Secret")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		_ = json.NewEncoder(w).Encode(dto.InitiatePaymentResponse{
			PaymentID: "pay_123", PaymentURL: "https://mock/pay/123", Amount: 15, Currency: "TRY",
		})
	})

	resp, err := c.InitiatePayment(context.Background(), dto.InitiatePaymentRequest{
		ReferenceID: "res_1", Amount: 15, Currency: "TRY", StudentID: "stu_1",
	})
	require.NoError(t, err)

	assert.Equal(t, "pay_123", resp.PaymentID)
	assert.Equal(t, "res_1", got.ReferenceID)
	assert.Equal(t, "/internal/payments/initiate", gotPath)
	assert.Equal(t, "test-secret", gotSecret)
}

func TestHTTPPaymentClient_RequestRefund_ReturnsRefund(t *testing.T) {
	c := newPaymentTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/payments/refund", r.URL.Path)
		_ = json.NewEncoder(w).Encode(dto.RefundResponse{RefundID: "ref_9", Status: "completed"})
	})

	resp, err := c.RequestRefund(context.Background(), dto.RefundRequest{ReferenceID: "res_1", Amount: 15, Currency: "TRY"})
	require.NoError(t, err)

	assert.Equal(t, "ref_9", resp.RefundID)
	assert.Equal(t, "completed", resp.Status)
}

func TestHTTPPaymentClient_InitiatePayment_ServerErrorPropagates(t *testing.T) {
	c := newPaymentTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.InitiatePayment(context.Background(), dto.InitiatePaymentRequest{ReferenceID: "res_1"})
	require.Error(t, err)
	assert.ErrorIs(t, err, client.ErrUnavailable, "a failed payment must never look like a success")
}
