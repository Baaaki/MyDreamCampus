package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/config"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// isUniqueViolation returns true if err is a Postgres unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ClosedDaysReader is the slice of ClosedDaysRepository that the reservation
// hot path needs. It is an interface so that a caching wrapper can be swapped
// in transparently.
type ClosedDaysReader interface {
	IsDateClosed(ctx context.Context, date pgtype.Date) (bool, error)
}

type ReservationService struct {
	reservationRepo  *repository.ReservationRepository
	cafeteriaRepo    *repository.CafeteriaRepository
	studentCacheRepo *repository.StudentCacheRepository
	closedDaysRepo   ClosedDaysReader
	paymentClient    PaymentClient
	cfg              *config.Config
	logger           *zap.Logger
}

func NewReservationService(
	reservationRepo *repository.ReservationRepository,
	cafeteriaRepo *repository.CafeteriaRepository,
	studentCacheRepo *repository.StudentCacheRepository,
	closedDaysRepo ClosedDaysReader,
	paymentClient PaymentClient,
	cfg *config.Config,
	logger *zap.Logger,
) *ReservationService {
	return &ReservationService{
		reservationRepo:  reservationRepo,
		cafeteriaRepo:    cafeteriaRepo,
		studentCacheRepo: studentCacheRepo,
		closedDaysRepo:   closedDaysRepo,
		paymentClient:    paymentClient,
		cfg:              cfg,
		logger:           logger,
	}
}

// CreateReservation creates a single reservation
func (s *ReservationService) CreateReservation(ctx context.Context, studentID uuid.UUID, req dto.CreateReservationRequest) (*dto.CreateReservationResponse, error) {
	// 1. Check if student is active
	student, err := s.studentCacheRepo.GetStudentCacheByID(ctx, studentID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFoundRepo) {
			return nil, sharedErrors.ErrNotFound
		}
		s.logger.Error("failed to get student cache", zap.Error(err), zap.String("student_id", studentID.String()))
		return nil, err
	}

	if !student.IsActive {
		return nil, serviceErrors.ErrStudentDeactivated
	}

	// 2. Parse and validate date
	reservationDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, sharedErrors.ErrBadRequest
	}

	// 3. Check the date is open (not a closed day)
	if err := s.validateReservationDate(ctx, reservationDate); err != nil {
		return nil, err
	}

	// 4. Parse cafeteria ID
	cafeteriaID, err := uuid.Parse(req.CafeteriaID)
	if err != nil {
		return nil, sharedErrors.ErrBadRequest
	}

	// 6. Get and validate cafeteria
	cafeteria, err := s.cafeteriaRepo.GetCafeteriaByID(ctx, cafeteriaID)
	if err != nil {
		if errors.Is(err, serviceErrors.ErrCafeteriaNotFoundRepo) {
			return nil, serviceErrors.ErrCafeteriaNotFound
		}
		return nil, err
	}

	if !cafeteria.IsActive {
		return nil, serviceErrors.ErrCafeteriaNotActive
	}

	// 7. Validate meal time and menu type
	if err := s.validateMealTimeAndMenu(req.MealTime, req.MenuType, cafeteria); err != nil {
		return nil, err
	}

	// 8. Check for active reservation (conflict)
	mealTimeEnum, _ := s.parseMealTimeEnum(req.MealTime)
	existing, err := s.reservationRepo.CheckActiveReservation(ctx, db.CheckActiveReservationParams{
		StudentID:       utils.UUIDToPgtype(studentID),
		ReservationDate: pgtype.Date{Time: reservationDate, Valid: true},
		MealTime:        mealTimeEnum,
	})
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, serviceErrors.ErrActiveReservationExists
	}

	// 9. Create reservation as pending (before calling payment).
	//    The partial unique index on (student_id, date, meal_time) WHERE status IN ('pending','confirmed')
	//    is the source of truth for duplicate prevention.
	menuTypeEnum, _ := s.parseMenuTypeEnum(req.MenuType)
	expiresAt := clock.Now().Add(time.Duration(s.cfg.Reservation.TimeoutMinutes) * time.Minute)

	reservation, err := s.reservationRepo.CreateReservation(ctx, db.CreateReservationParams{
		BatchID:         pgtype.UUID{Valid: false},
		StudentID:       utils.UUIDToPgtype(studentID),
		CafeteriaID:     utils.UUIDToPgtype(cafeteriaID),
		ReservationDate: pgtype.Date{Time: reservationDate, Valid: true},
		MealTime:        mealTimeEnum,
		MenuType:        menuTypeEnum,
		Status:          db.MealReservationStatusEnumPending,
		ExpiresAt:       pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, serviceErrors.ErrActiveReservationExists
		}
		s.logger.Error("failed to create pending reservation", zap.Error(err))
		return nil, err
	}

	// 10. Initiate payment. On failure, compensate by cancelling the pending row.
	referenceID := fmt.Sprintf("res_%s", reservation.ID.String())
	paymentResp, err := s.paymentClient.InitiatePayment(ctx, dto.InitiatePaymentRequest{
		ReferenceID: referenceID,
		Amount:      s.cfg.Reservation.MealPriceTRY,
		Currency:    "TRY",
		Description: fmt.Sprintf("Meal reservation - %s %s", req.Date, req.MealTime),
		StudentID:   studentID.String(),
	})
	if err != nil {
		s.logger.Error("payment initiation failed, rolling back pending reservation",
			zap.Error(err), zap.String("reservation_id", reservation.ID.String()))
		if _, rbErr := s.reservationRepo.UpdateReservationByID(ctx, db.UpdateReservationByIDParams{
			ID:        reservation.ID,
			Status:    db.MealReservationStatusEnumCancelled,
			ExpiresAt: pgtype.Timestamptz{Valid: false},
		}); rbErr != nil {
			s.logger.Error("rollback failed, pending row will be expired by worker",
				zap.Error(rbErr), zap.String("reservation_id", reservation.ID.String()))
		}
		return nil, err
	}

	s.logger.Info("pending reservation created, awaiting payment.completed event",
		zap.String("reservation_id", reservation.ID.String()),
		zap.String("student_id", studentID.String()),
		zap.String("payment_id", paymentResp.PaymentID),
		zap.String("date", req.Date),
		zap.String("meal_time", req.MealTime),
	)

	return &dto.CreateReservationResponse{
		ReservationID: reservation.ID.String(),
		PaymentURL:    paymentResp.PaymentURL,
		Amount:        s.cfg.Reservation.MealPriceTRY,
		Currency:      "TRY",
		ExpiresAt:     expiresAt,
		Reservation: dto.ReservationResponse{
			ID:            reservation.ID.String(),
			Date:          req.Date,
			MealTime:      req.MealTime,
			MenuType:      req.MenuType,
			CafeteriaName: cafeteria.Name,
			Status:        string(reservation.Status),
			IsUsed:        reservation.IsUsed,
			CreatedAt:     reservation.CreatedAt.Time,
		},
	}, nil
}

// CreateBatchReservation creates multiple reservations atomically
func (s *ReservationService) CreateBatchReservation(ctx context.Context, studentID uuid.UUID, req dto.BatchReservationRequest) (*dto.CreateBatchReservationResponse, error) {
	// 1. Check if student is active
	student, err := s.studentCacheRepo.GetStudentCacheByID(ctx, studentID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFoundRepo) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, err
	}

	if !student.IsActive {
		return nil, serviceErrors.ErrStudentDeactivated
	}

	// 2. Validate all reservations (collect all errors)
	validationErrors := make([]dto.ValidationError, 0)
	conflicts := make([]dto.ReservationConflict, 0)

	reservationParams := make([]db.CreateReservationParams, 0, len(req.Reservations))
	cafeterias := make(map[string]db.Cafeteria)

	batchID := uuid.New()
	expiresAt := clock.Now().Add(time.Duration(s.cfg.Reservation.TimeoutMinutes) * time.Minute)

	for i, r := range req.Reservations {
		// Parse date
		reservationDate, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			validationErrors = append(validationErrors, dto.ValidationError{
				Index:    i,
				Date:     r.Date,
				MealTime: r.MealTime,
				Code:     "INVALID_DATE_FORMAT",
				Message:  "Invalid date format (expected YYYY-MM-DD)",
			})
			continue
		}

		// Validate date
		if err := s.validateReservationDate(ctx, reservationDate); err != nil {
			validationErrors = append(validationErrors, dto.ValidationError{
				Index:    i,
				Date:     r.Date,
				MealTime: r.MealTime,
				Code:     "INVALID_DATE_RANGE",
				Message:  err.Error(),
			})
			continue
		}

		// Parse cafeteria ID
		cafeteriaID, err := uuid.Parse(r.CafeteriaID)
		if err != nil {
			validationErrors = append(validationErrors, dto.ValidationError{
				Index:    i,
				Date:     r.Date,
				MealTime: r.MealTime,
				Code:     "INVALID_CAFETERIA_ID",
				Message:  "Invalid cafeteria ID format",
			})
			continue
		}

		// Get cafeteria (cache it)
		var cafeteria db.Cafeteria
		if cached, ok := cafeterias[r.CafeteriaID]; ok {
			cafeteria = cached
		} else {
			cafeteria, err = s.cafeteriaRepo.GetCafeteriaByID(ctx, cafeteriaID)
			if err != nil {
				if errors.Is(err, serviceErrors.ErrCafeteriaNotFoundRepo) {
					validationErrors = append(validationErrors, dto.ValidationError{
						Index:    i,
						Date:     r.Date,
						MealTime: r.MealTime,
						Code:     "CAFETERIA_NOT_FOUND",
						Message:  "Cafeteria not found",
					})
				} else {
					return nil, err
				}
				continue
			}
			cafeterias[r.CafeteriaID] = cafeteria
		}

		if !cafeteria.IsActive {
			validationErrors = append(validationErrors, dto.ValidationError{
				Index:    i,
				Date:     r.Date,
				MealTime: r.MealTime,
				Code:     "CAFETERIA_NOT_ACTIVE",
				Message:  "Cafeteria is not active",
			})
			continue
		}

		// Validate meal time and menu
		if err := s.validateMealTimeAndMenu(r.MealTime, r.MenuType, cafeteria); err != nil {
			validationErrors = append(validationErrors, dto.ValidationError{
				Index:    i,
				Date:     r.Date,
				MealTime: r.MealTime,
				Code:     s.getErrorCode(err),
				Message:  err.Error(),
			})
			continue
		}

		// Stage param; conflicts are resolved in one query after the loop.
		mealTimeEnum, _ := s.parseMealTimeEnum(r.MealTime)
		menuTypeEnum, _ := s.parseMenuTypeEnum(r.MenuType)
		reservationParams = append(reservationParams, db.CreateReservationParams{
			BatchID:         pgtype.UUID{Bytes: batchID, Valid: true},
			StudentID:       utils.UUIDToPgtype(studentID),
			CafeteriaID:     utils.UUIDToPgtype(cafeteriaID),
			ReservationDate: pgtype.Date{Time: reservationDate, Valid: true},
			MealTime:        mealTimeEnum,
			MenuType:        menuTypeEnum,
			Status:          db.MealReservationStatusEnumPending, // Confirmed by payment.completed event
			ExpiresAt:       pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
	}

	// If there are any validation errors, return them
	if len(validationErrors) > 0 {
		s.logger.Error("batch reservation validation failed", zap.Any("errors", validationErrors))
		return nil, fmt.Errorf("%w", serviceErrors.ErrValidationErrors)
	}

	// Batch conflict check in a single query. Replaces the previous N+1 probe.
	if len(reservationParams) > 0 {
		dates := make([]pgtype.Date, 0, len(reservationParams))
		mealTimes := make([]string, 0, len(reservationParams))
		for _, p := range reservationParams {
			dates = append(dates, p.ReservationDate)
			mealTimes = append(mealTimes, string(p.MealTime))
		}

		existingRows, err := s.reservationRepo.CheckActiveReservationsForSlots(ctx, db.CheckActiveReservationsForSlotsParams{
			StudentID: utils.UUIDToPgtype(studentID),
			Dates:     dates,
			MealTimes: mealTimes,
		})
		if err != nil {
			return nil, err
		}

		for _, row := range existingRows {
			conflicts = append(conflicts, dto.ReservationConflict{
				Date:                  row.ReservationDate.Time.Format("2006-01-02"),
				MealTime:              string(row.MealTime),
				ExistingReservationID: row.ID.String(),
				CafeteriaName:         row.CafeteriaName,
				Status:                string(row.Status),
			})
		}
	}

	// If there are any conflicts, return them
	if len(conflicts) > 0 {
		s.logger.Error("batch reservation conflicts found", zap.Any("conflicts", conflicts))
		return nil, fmt.Errorf("%w", serviceErrors.ErrReservationConflicts)
	}

	// Persist as pending FIRST (one transaction). The partial unique index on
	// (student_id, date, meal_time) WHERE status IN ('pending','confirmed') is the
	// authoritative duplicate guard.
	reservations, err := s.reservationRepo.CreateBatchReservations(ctx, reservationParams)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, serviceErrors.ErrReservationConflicts
		}
		s.logger.Error("failed to create pending batch reservations", zap.Error(err))
		return nil, err
	}

	// Initiate payment. On failure, cancel the whole batch.
	totalAmount := s.cfg.Reservation.MealPriceTRY * float64(len(reservationParams))
	referenceID := fmt.Sprintf("bat_%s", batchID.String())
	paymentResp, err := s.paymentClient.InitiatePayment(ctx, dto.InitiatePaymentRequest{
		ReferenceID: referenceID,
		Amount:      totalAmount,
		Currency:    "TRY",
		Description: fmt.Sprintf("Batch meal reservation - %d meals", len(reservationParams)),
		StudentID:   studentID.String(),
	})
	if err != nil {
		s.logger.Error("payment initiation failed for batch, rolling back pending rows",
			zap.Error(err), zap.String("batch_id", batchID.String()))
		if rbErr := s.reservationRepo.UpdateReservationsByBatchID(ctx, db.UpdateReservationsByBatchIDParams{
			BatchID:   pgtype.UUID{Bytes: batchID, Valid: true},
			Status:    db.MealReservationStatusEnumCancelled,
			ExpiresAt: pgtype.Timestamptz{Valid: false},
		}); rbErr != nil {
			s.logger.Error("batch rollback failed, rows will be expired by worker",
				zap.Error(rbErr), zap.String("batch_id", batchID.String()))
		}
		return nil, err
	}

	// Build response
	reservationResponses := make([]dto.ReservationResponse, 0, len(reservations))
	for _, r := range reservations {
		cafeteria := cafeterias[r.CafeteriaID.String()]
		reservationResponses = append(reservationResponses, dto.ReservationResponse{
			ID:            r.ID.String(),
			Date:          r.ReservationDate.Time.Format("2006-01-02"),
			MealTime:      string(r.MealTime),
			MenuType:      string(r.MenuType),
			CafeteriaName: cafeteria.Name,
			Status:        string(r.Status),
			IsUsed:        r.IsUsed,
			CreatedAt:     r.CreatedAt.Time,
		})
	}

	s.logger.Info("batch reservation created after successful payment",
		zap.String("batch_id", batchID.String()),
		zap.String("student_id", studentID.String()),
		zap.String("payment_id", paymentResp.PaymentID),
		zap.Int("count", len(reservations)),
	)

	return &dto.CreateBatchReservationResponse{
		BatchID:      batchID.String(),
		PaymentURL:   paymentResp.PaymentURL,
		TotalAmount:  totalAmount,
		Currency:     "TRY",
		ExpiresAt:    expiresAt,
		Reservations: reservationResponses,
	}, nil
}

// GetMyReservations returns student's reservations with optional filters
func (s *ReservationService) GetMyReservations(ctx context.Context, studentID uuid.UUID, query dto.MyReservationsQuery) (*dto.MyReservationsResponse, error) {
	var reservations []db.GetStudentReservationsFilteredRow
	var err error

	// Parse optional filters
	var fromDate, toDate pgtype.Date
	var status db.NullReservationStatusEnum

	if query.FromDate != "" {
		t, err := time.Parse("2006-01-02", query.FromDate)
		if err != nil {
			return nil, sharedErrors.ErrBadRequest
		}
		fromDate = pgtype.Date{Time: t, Valid: true}
	}

	if query.ToDate != "" {
		t, err := time.Parse("2006-01-02", query.ToDate)
		if err != nil {
			return nil, sharedErrors.ErrBadRequest
		}
		toDate = pgtype.Date{Time: t, Valid: true}
	}

	if query.Status != "" {
		statusEnum, err := s.parseStatusEnum(query.Status)
		if err != nil {
			return nil, err
		}
		status = db.NullReservationStatusEnum{ReservationStatusEnum: statusEnum, Valid: true}
	}

	// Handle pagination
	var limit int32 = 0
	var offset int32 = 0
	var totalCount int64 = 0

	if query.Page > 0 {
		if query.Limit > 0 {
			limit = int32(query.Limit)
		} else {
			limit = 10 // default limit
		}
		offset = int32((query.Page - 1)) * limit

		// Get total count for pagination
		totalCount, err = s.reservationRepo.CountStudentReservationsFiltered(ctx, db.CountStudentReservationsFilteredParams{
			StudentID: utils.UUIDToPgtype(studentID),
			Column2:   fromDate,
			Column3:   toDate,
			Status:    status,
		})
		if err != nil {
			s.logger.Error("failed to count student reservations", zap.Error(err))
			return nil, err
		}
	}

	reservations, err = s.reservationRepo.GetStudentReservationsFiltered(ctx, db.GetStudentReservationsFilteredParams{
		StudentID: utils.UUIDToPgtype(studentID),
		Column2:   fromDate,
		Column3:   toDate,
		Status:    status,
		Column4:   limit,
		Offset:    offset,
	})
	if err != nil {
		s.logger.Error("failed to get student reservations", zap.Error(err))
		return nil, err
	}

	// Build response
	response := &dto.MyReservationsResponse{
		Reservations: make([]dto.ReservationResponse, 0, len(reservations)),
		Summary: dto.ReservationSummary{
			Total:     len(reservations),
			Confirmed: 0,
			Pending:   0,
			Used:      0,
			Cancelled: 0,
		},
	}

	// Add pagination info if requested
	if query.Page > 0 {
		totalPages := int(totalCount) / int(limit)
		if int(totalCount)%int(limit) > 0 {
			totalPages++
		}
		response.Pagination = &dto.PaginationInfo{
			Page:       query.Page,
			Limit:      int(limit),
			TotalItems: int(totalCount),
			TotalPages: totalPages,
		}
		response.Summary.Total = int(totalCount)
	}

	for _, r := range reservations {
		response.Reservations = append(response.Reservations, dto.ReservationResponse{
			ID:       r.ID.String(),
			Date:     r.ReservationDate.Time.Format("2006-01-02"),
			MealTime: string(r.MealTime),
			MenuType: string(r.MenuType),
			Cafeteria: &dto.CafeteriaInfo{
				ID:       r.CafeteriaID.String(),
				Name:     r.CafeteriaName,
				Location: r.CafeteriaLocation,
			},
			Status:    string(r.Status),
			IsUsed:    r.IsUsed,
			CreatedAt: r.CreatedAt.Time,
		})

		// Update summary
		switch r.Status {
		case db.MealReservationStatusEnumConfirmed:
			response.Summary.Confirmed++
			if r.IsUsed {
				response.Summary.Used++
			}
		case db.MealReservationStatusEnumPending:
			response.Summary.Pending++
		case db.MealReservationStatusEnumCancelled:
			response.Summary.Cancelled++
		}
	}

	return response, nil
}

// CancelReservation cancels a confirmed reservation and processes refund synchronously
func (s *ReservationService) CancelReservation(ctx context.Context, studentID uuid.UUID, reservationID string) (*dto.CancelReservationResponse, error) {
	// 1. Parse reservation ID
	resID, err := uuid.Parse(reservationID)
	if err != nil {
		return nil, sharedErrors.ErrBadRequest
	}

	// 2. Check if student is active
	student, err := s.studentCacheRepo.GetStudentCacheByID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	if !student.IsActive {
		return nil, serviceErrors.ErrStudentDeactivated
	}

	// 3. Get reservation
	reservation, err := s.reservationRepo.GetReservationByID(ctx, resID)
	if err != nil {
		if errors.Is(err, serviceErrors.ErrReservationNotFoundRepo) {
			return nil, serviceErrors.ErrReservationNotFound
		}
		return nil, err
	}

	// 5. Check ownership
	if utils.PgtypeToUUID(reservation.StudentID) != studentID {
		return nil, serviceErrors.ErrNotOwner
	}

	// 6. Check status (only confirmed can be cancelled)
	if reservation.Status != db.MealReservationStatusEnumConfirmed {
		return nil, serviceErrors.ErrInvalidStatusForCancel
	}

	// 7. Check if already used
	if reservation.IsUsed {
		return nil, serviceErrors.ErrReservationAlreadyUsed
	}

	// 8. Enforce the cancellation cut-off: meals cannot be cancelled once the
	//    cut-off window before the meal's start has passed.
	if err := s.validateCancelCutoff(reservation.ReservationDate.Time, reservation.MealTime); err != nil {
		return nil, err
	}

	// 9. Request refund synchronously
	refundResp, err := s.paymentClient.RequestRefund(ctx, dto.RefundRequest{
		ReferenceID: resID.String(),
		Amount:      s.cfg.Reservation.MealPriceTRY,
		Currency:    "TRY",
		Reason:      "Student cancelled reservation",
	})
	if err != nil {
		s.logger.Error("refund failed", zap.Error(err), zap.String("reservation_id", reservationID))
		return nil, err
	}

	// 9. Cancel reservation and create outbox event
	eventPayload := map[string]any{
		"reservation_id": reservation.ID.String(),
		"student_id":     student.ID.String(),
		"student_number": student.StudentNumber,
		"date":           reservation.ReservationDate.Time.Format("2006-01-02"),
		"meal_time":      string(reservation.MealTime),
		"refund_amount":  s.cfg.Reservation.MealPriceTRY,
		"currency":       "TRY",
	}

	_, err = s.reservationRepo.CancelReservationWithRefund(ctx, resID, eventPayload)
	if err != nil {
		s.logger.Error("failed to cancel reservation", zap.Error(err))
		return nil, err
	}

	s.logger.Info("reservation cancelled",
		zap.String("reservation_id", reservationID),
		zap.String("student_id", studentID.String()),
	)

	return &dto.CancelReservationResponse{
		ReservationID: reservationID,
		RefundAmount:  s.cfg.Reservation.MealPriceTRY,
		Currency:      "TRY",
		RefundStatus:  refundResp.Status,
	}, nil
}

// UseReservation validates QR and marks reservation as used
func (s *ReservationService) UseReservation(ctx context.Context, studentID uuid.UUID, req dto.UseReservationRequest) (*dto.UseReservationResponse, error) {
	// 1. Parse QR payload
	cafeteriaID, date, mealTime, window, signature, err := s.parseQRPayload(req.QRPayload)
	if err != nil {
		return nil, serviceErrors.ErrInvalidQR
	}

	// 2. Verify signature (and that the rotating window is still fresh)
	if !s.verifyQRSignature(cafeteriaID, date, mealTime, window, signature) {
		return nil, serviceErrors.ErrInvalidQR
	}

	// 3. Validate date (must be today)
	today := clock.Now().In(time.FixedZone("UTC+3", 3*3600)).Format("2006-01-02")
	if date != today {
		return nil, serviceErrors.ErrInvalidQRDate
	}

	// 4. Validate time window
	if err := s.validateMealTimeWindow(mealTime); err != nil {
		return nil, err
	}

	// 5. Check if student is active
	student, err := s.studentCacheRepo.GetStudentCacheByID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	if !student.IsActive {
		return nil, serviceErrors.ErrStudentDeactivated
	}

	// 6. Find reservation
	parsedCafeteriaID, _ := uuid.Parse(cafeteriaID)
	parsedDate, _ := time.Parse("2006-01-02", date)
	mealTimeEnum, _ := s.parseMealTimeEnum(mealTime)

	reservation, err := s.reservationRepo.FindReservationForQR(ctx, db.FindReservationForQRParams{
		CafeteriaID:     utils.UUIDToPgtype(parsedCafeteriaID),
		ReservationDate: pgtype.Date{Time: parsedDate, Valid: true},
		MealTime:        mealTimeEnum,
		StudentID:       utils.UUIDToPgtype(studentID),
	})
	if err != nil {
		if errors.Is(err, serviceErrors.ErrReservationNotFoundRepo) {
			return nil, serviceErrors.ErrNoReservation
		}
		return nil, err
	}

	// 7. Mark as used
	_, err = s.reservationRepo.MarkReservationUsed(ctx, utils.PgtypeToUUID(reservation.ID))
	if err != nil {
		s.logger.Error("failed to mark reservation as used", zap.Error(err))
		return nil, err
	}

	s.logger.Info("reservation used",
		zap.String("reservation_id", reservation.ID.String()),
		zap.String("student_id", studentID.String()),
		zap.String("cafeteria", reservation.CafeteriaName),
	)

	return &dto.UseReservationResponse{
		Message:       "Reservation validated",
		ReservationID: reservation.ID.String(),
		CafeteriaName: reservation.CafeteriaName,
		MealTime:      mealTime,
		MenuType:      string(reservation.MenuType),
	}, nil
}

// GenerateQR generates QR payload for cafeteria (Admin only)
func (s *ReservationService) GenerateQR(ctx context.Context, cafeteriaID string, date string, mealTime string) (*dto.QRResponse, error) {
	// Parse cafeteria ID
	cID, err := uuid.Parse(cafeteriaID)
	if err != nil {
		return nil, sharedErrors.ErrBadRequest
	}

	// Get cafeteria
	cafeteria, err := s.cafeteriaRepo.GetCafeteriaByID(ctx, cID)
	if err != nil {
		if errors.Is(err, serviceErrors.ErrCafeteriaNotFoundRepo) {
			return nil, serviceErrors.ErrCafeteriaNotFound
		}
		return nil, err
	}

	// Default date to today if not provided
	if date == "" {
		date = clock.Now().In(time.FixedZone("UTC+3", 3*3600)).Format("2006-01-02")
	}

	// Generate QR payload
	qrPayload := s.generateQRPayload(cafeteriaID, date, mealTime)

	// Determine time window
	var startHour, endHour int
	if mealTime == "lunch" {
		startHour = s.cfg.MealTime.LunchStartHour
		endHour = s.cfg.MealTime.LunchEndHour
	} else {
		startHour = s.cfg.MealTime.DinnerStartHour
		endHour = s.cfg.MealTime.DinnerEndHour
	}

	return &dto.QRResponse{
		CafeteriaID:   cafeteriaID,
		CafeteriaName: cafeteria.Name,
		Date:          date,
		MealTime:      mealTime,
		QRPayload:     qrPayload,
		ValidTimeWindow: dto.ValidTimeWindow{
			Start: fmt.Sprintf("%02d:00", startHour),
			End:   fmt.Sprintf("%02d:00", endHour),
		},
	}, nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (s *ReservationService) validateReservationDate(ctx context.Context, date time.Time) error {
	isClosed, err := s.closedDaysRepo.IsDateClosed(ctx, pgtype.Date{Time: date, Valid: true})
	if err != nil {
		s.logger.Error("failed to check closed day", zap.Error(err), zap.Time("date", date))
		return err
	}
	if isClosed {
		return serviceErrors.ErrCafeteriaClosedOnDate
	}

	return nil
}

func (s *ReservationService) validateMealTimeAndMenu(mealTime, menuType string, cafeteria db.Cafeteria) error {
	if mealTime != "lunch" && mealTime != "dinner" {
		return serviceErrors.ErrInvalidMealTime
	}

	if menuType != "normal" && menuType != "vegan" {
		return serviceErrors.ErrInvalidMenuType
	}

	if mealTime == "dinner" && !cafeteria.ServesDinner {
		return serviceErrors.ErrCafeteriaNoDinner
	}

	if menuType == "vegan" && !cafeteria.HasVeganMenu {
		return serviceErrors.ErrCafeteriaNoVegan
	}

	return nil
}

// validateCancelCutoff rejects cancellations submitted after the configured
// cut-off window before the meal's scheduled start time (UTC+3).
func (s *ReservationService) validateCancelCutoff(reservationDate time.Time, mealTime db.MealTimeEnum) error {
	loc := time.FixedZone("UTC+3", 3*3600)

	var mealStartHour int
	switch mealTime {
	case db.MealMealTimeEnumLunch:
		mealStartHour = s.cfg.MealTime.LunchStartHour
	case db.MealMealTimeEnumDinner:
		mealStartHour = s.cfg.MealTime.DinnerStartHour
	default:
		return serviceErrors.ErrInvalidMealTime
	}

	mealStart := time.Date(
		reservationDate.Year(), reservationDate.Month(), reservationDate.Day(),
		mealStartHour, 0, 0, 0, loc,
	)
	cutoff := mealStart.Add(-time.Duration(s.cfg.Reservation.CancelCutoffHours) * time.Hour)

	if clock.Now().In(loc).After(cutoff) {
		return serviceErrors.ErrCancelCutoffPassed
	}
	return nil
}

func (s *ReservationService) validateMealTimeWindow(mealTime string) error {
	now := clock.Now().In(time.FixedZone("UTC+3", 3*3600))
	hour := now.Hour()

	if mealTime == "lunch" {
		if hour < s.cfg.MealTime.LunchStartHour || hour >= s.cfg.MealTime.LunchEndHour {
			return serviceErrors.ErrOutsideMealTimeWindow
		}
	} else if mealTime == "dinner" {
		if hour < s.cfg.MealTime.DinnerStartHour || hour >= s.cfg.MealTime.DinnerEndHour {
			return serviceErrors.ErrOutsideMealTimeWindow
		}
	}

	return nil
}

// qrWindow returns the current rotating window bucket for QR signatures.
// A new bucket starts every QRValidityWindowSeconds, which shortens the
// blast radius if a QR image leaks out of the cafeteria.
func (s *ReservationService) qrWindow() int64 {
	return clock.Now().Unix() / int64(s.cfg.Reservation.QRValidityWindowSeconds)
}

func (s *ReservationService) generateQRPayload(cafeteriaID, date, mealTime string) string {
	window := s.qrWindow()
	payload := fmt.Sprintf("%s:%s:%s:%d", cafeteriaID, date, mealTime, window)
	signature := s.signQRPayload(payload)
	return fmt.Sprintf("%s:%s", payload, signature)
}

func (s *ReservationService) signQRPayload(payload string) string {
	h := hmac.New(sha256.New, []byte(s.cfg.QR.Secret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *ReservationService) parseQRPayload(qrPayload string) (cafeteriaID, date, mealTime, window, signature string, err error) {
	parts := strings.Split(qrPayload, ":")
	if len(parts) != 5 {
		return "", "", "", "", "", fmt.Errorf("invalid QR format")
	}
	return parts[0], parts[1], parts[2], parts[3], parts[4], nil
}

// verifyQRSignature accepts the current bucket or the immediately previous
// bucket so that a scan at a bucket boundary is not rejected.
func (s *ReservationService) verifyQRSignature(cafeteriaID, date, mealTime, window, signature string) bool {
	parsed, err := strconv.ParseInt(window, 10, 64)
	if err != nil {
		return false
	}
	current := s.qrWindow()
	if parsed != current && parsed != current-1 {
		return false
	}
	payload := fmt.Sprintf("%s:%s:%s:%s", cafeteriaID, date, mealTime, window)
	expected := s.signQRPayload(payload)
	return hmac.Equal([]byte(signature), []byte(expected))
}

func (s *ReservationService) parseMealTimeEnum(mealTime string) (db.MealTimeEnum, error) {
	switch mealTime {
	case "lunch":
		return db.MealMealTimeEnumLunch, nil
	case "dinner":
		return db.MealMealTimeEnumDinner, nil
	default:
		return "", serviceErrors.ErrInvalidMealTime
	}
}

func (s *ReservationService) parseMenuTypeEnum(menuType string) (db.MenuTypeEnum, error) {
	switch menuType {
	case "normal":
		return db.MealMenuTypeEnumNormal, nil
	case "vegan":
		return db.MealMenuTypeEnumVegan, nil
	default:
		return "", serviceErrors.ErrInvalidMenuType
	}
}

func (s *ReservationService) parseStatusEnum(status string) (db.ReservationStatusEnum, error) {
	switch status {
	case "pending":
		return db.MealReservationStatusEnumPending, nil
	case "confirmed":
		return db.MealReservationStatusEnumConfirmed, nil
	case "cancelled":
		return db.MealReservationStatusEnumCancelled, nil
	case "expired":
		return db.MealReservationStatusEnumExpired, nil
	default:
		return "", sharedErrors.ErrBadRequest
	}
}

func (s *ReservationService) getErrorCode(err error) string {
	switch {
	case errors.Is(err, serviceErrors.ErrCafeteriaNoDinner):
		return "CAFETERIA_NO_DINNER"
	case errors.Is(err, serviceErrors.ErrCafeteriaNoVegan):
		return "CAFETERIA_NO_VEGAN"
	case errors.Is(err, serviceErrors.ErrInvalidMealTime):
		return "INVALID_MEAL_TIME"
	case errors.Is(err, serviceErrors.ErrInvalidMenuType):
		return "INVALID_MENU_TYPE"
	default:
		return "VALIDATION_ERROR"
	}
}
