package application

import (
	"context"
	"errors"
	"fmt"
	"grpc-course/internal/payment/domain"
	"log/slog"

	"github.com/google/uuid"
)

type PaymentService struct {
	repo   domain.PaymentRepository
	logger *slog.Logger
}

func NewPaymentService(
	repo domain.PaymentRepository,
	logger *slog.Logger,
) *PaymentService {
	return &PaymentService{
		repo:   repo,
		logger: logger,
	}
}

func (s *PaymentService) CreatePayment(
	ctx context.Context,
	orderID string,
	amountMinor int64,
	currency string,
) (*domain.Payment, error) {

	got, err := s.repo.GetByOrderID(ctx, orderID)
	if err == nil {
		return got, nil
	}
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		return nil, err
	}

	paymentID := uuid.NewString()
	payment, err := domain.NewPayment(paymentID, orderID, amountMinor, currency)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	err = s.repo.CreatePayment(ctx, payment)
	if err == nil {
		s.logger.Info(
			"payment created",
			"order_id", orderID,
			"amount_minor", amountMinor,
			"currency", currency,
		)

		return payment, nil
	}
	if errors.Is(err, domain.ErrPaymentAlreadyExists) {
		existingPayment, getErr := s.repo.GetByOrderID(ctx, orderID)
		if getErr != nil {
			return nil, fmt.Errorf("get payment after concurrent creation: %w", getErr)
		}
		return existingPayment, nil
	}

	return nil, fmt.Errorf("create payment: %w", err)
}
