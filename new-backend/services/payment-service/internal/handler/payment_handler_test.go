package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	"github.com/baaaki/mydreamcampus/payment/internal/service"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// Fixed ids so no uuid can contain the card digits the log test searches for.
var (
	studentID = uuid.MustParse("0190a1b2-0000-7000-8000-00000000000a")
	paymentID = uuid.MustParse("0190a1b2-0000-7000-8000-00000000000b")
)

// stubStore holds one payment. The embedded interface leaves the methods
// these tests never reach unimplemented.
type stubStore struct {
	service.Store
	payment db.Payment
}

func (s *stubStore) GetPaymentByID(_ context.Context, id uuid.UUID) (db.Payment, error) {
	if id != utils.PgtypeToUUID(s.payment.ID) {
		return db.Payment{}, serviceErrors.ErrPaymentNotFoundRepo
	}
	return s.payment, nil
}

func (s *stubStore) CompletePaymentWithEvent(_ context.Context, p db.CompletePaymentParams, _ map[string]any) (db.Payment, error) {
	s.payment.Status = db.PaymentPaymentStatusEnumCompleted
	s.payment.CardBrand, s.payment.CardLast4, s.payment.CompletedAt = p.CardBrand, p.CardLast4, p.CompletedAt
	return s.payment, nil
}

func (s *stubStore) FailPaymentWithEvent(_ context.Context, p db.FailPaymentParams, _ map[string]any) (db.Payment, error) {
	s.payment.Status = db.PaymentPaymentStatusEnumFailed
	s.payment.CardBrand, s.payment.CardLast4, s.payment.FailureReason = p.CardBrand, p.CardLast4, p.FailureReason
	return s.payment, nil
}

func (s *stubStore) ExpirePayment(context.Context, uuid.UUID) error { return nil }

// setup wires the handler behind a stand-in for the auth chain and routes
// every log line, the handler's and the service's, into one observer.
func setup(t *testing.T) (*gin.Engine, *observer.ObservedLogs) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zap.DebugLevel)
	prev := logger.Log
	logger.Log = zap.New(core)
	t.Cleanup(func() { logger.Log = prev })

	store := &stubStore{payment: db.Payment{
		ID:             utils.UUIDToPgtype(paymentID),
		ReferenceID:    "res_0190a1b2-0000-7000-8000-00000000000c",
		StudentID:      utils.UUIDToPgtype(studentID),
		Amount:         utils.Float64ToPgNumeric(45),
		RefundedAmount: utils.Float64ToPgNumeric(0),
		Currency:       "TRY",
		Status:         db.PaymentPaymentStatusEnumPending,
		ExpiresAt:      utils.TimeToPgTimestamptz(time.Now().Add(time.Hour)),
	}}
	svc := service.NewPaymentService(store, 15*time.Minute, logger.Log)

	r := gin.New()
	g := r.Group("/api/payments", func(c *gin.Context) { c.Set("user_id", studentID.String()) })
	NewPaymentHandler(svc).RegisterRoutes(g)
	return r, logs
}

func confirm(r *gin.Engine, id, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/payments/"+id+"/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func cardBody(number string) string {
	return fmt.Sprintf(`{"card_number":%q,"exp_month":12,"exp_year":30,"cvc":"7391","cardholder_name":"Zeynep Şahin"}`, number)
}

func TestConfirmPayment_Outcomes_ReturnExpectedStatus(t *testing.T) {
	cases := []struct {
		name, id, body string
		want           int
		status         string
	}{
		{"success card", paymentID.String(), cardBody("4242 4242 4242 4242"), http.StatusOK, "completed"},
		{"declined card", paymentID.String(), cardBody("4000000000000002"), http.StatusOK, "failed"},
		{"luhn fails", paymentID.String(), cardBody("4242424242424241"), http.StatusUnprocessableEntity, ""},
		{"malformed body", paymentID.String(), `{"card_number":4242424242424242}`, http.StatusBadRequest, ""},
		{"unknown payment", uuid.NewString(), cardBody("4242424242424242"), http.StatusNotFound, ""},
		{"malformed payment id", "not-a-uuid", cardBody("4242424242424242"), http.StatusNotFound, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := setup(t)

			w := confirm(r, tc.id, tc.body)

			assert.Equal(t, tc.want, w.Code, w.Body.String())
			if tc.status != "" {
				assert.Contains(t, w.Body.String(), `"status":"`+tc.status+`"`)
			}
		})
	}
}

// The card number and CVC reach the handler and the service; neither may
// leave them through a log line or the response, whatever the outcome.
func TestConfirmPayment_AnyOutcome_NeverLeaksCardData(t *testing.T) {
	secrets := []string{"4242424242424242", "4242 4242 4242 4242", "4000000000000002", "4242424242424241", "7391"}
	bodies := []string{
		cardBody("4242 4242 4242 4242"),
		cardBody("4000000000000002"),
		cardBody("4242424242424241"),
		`{"card_number":4242424242424242,"cvc":7391}`,
		`{"card_number":"4242424242424242","cvc":"7391"`,
	}
	for _, body := range bodies {
		r, logs := setup(t)

		w := confirm(r, paymentID.String(), body)

		for _, secret := range secrets {
			assert.NotContains(t, w.Body.String(), secret, "response for %s", body)
		}
		require.NotZero(t, logs.Len(), "expected the request to log something for %s", body)
		for _, entry := range logs.All() {
			line := entry.Message + fmt.Sprint(entry.ContextMap())
			for _, secret := range secrets {
				assert.NotContains(t, line, secret, "log line for %s", body)
			}
		}
	}
}

func TestGetPayment_Owner_ReturnsPaymentWithoutCardData(t *testing.T) {
	r, _ := setup(t)
	require.Equal(t, http.StatusOK, confirm(r, paymentID.String(), cardBody("4242424242424242")).Code)

	req := httptest.NewRequest(http.MethodGet, "/api/payments/"+paymentID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"card_last4":"4242"`)
	assert.NotContains(t, w.Body.String(), "4242424242424242")
}
