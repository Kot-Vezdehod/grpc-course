package application

import (
	"context"
	"errors"
	"grpc-course/internal/payment/domain"
	"grpc-course/internal/payment/mocks"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreatePayment_HappyPath(t *testing.T) {
	repo := mocks.NewPaymentRepository(t)
	service := NewPaymentService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	orderID := "order-001"

	repo.On("GetByOrderID", mock.Anything, orderID).
		Return(nil, domain.ErrPaymentNotFound)

	repo.On("CreatePayment", mock.Anything, mock.MatchedBy(func(payment *domain.Payment) bool {
		return payment.OrderID == orderID &&
			payment.Currency == "USD" &&
			payment.AmountMinor == 1000 &&
			payment.ID != ""
	}),
	).Return(nil)

	got, err := service.CreatePayment(
		context.Background(),
		orderID,
		1000,
		"USD",
	)
	require.NoError(t, err)
	require.Equal(t, orderID, got.OrderID)
	require.Equal(t, int64(1000), got.AmountMinor)
	require.Equal(t, "USD", got.Currency)
	require.NotEmpty(t, got.ID)
}

func TestCreatePayment_AlreadyExists(t *testing.T) {
	repo := mocks.NewPaymentRepository(t)
	service := NewPaymentService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	orderID := "order-001"
	existingPayment := &domain.Payment{
		ID:          "payment-001",
		OrderID:     orderID,
		Currency:    "USD",
		AmountMinor: 1000,
	}

	repo.On("GetByOrderID", mock.Anything, orderID).
		Return(existingPayment, nil)

	got, err := service.CreatePayment(
		context.Background(),
		orderID,
		1000,
		"USD",
	)
	require.NoError(t, err)
	require.Equal(t, existingPayment, got)

	repo.AssertNotCalled(t,
		"CreatePayment",
		mock.Anything,
		mock.Anything,
	)
}

func TestCreatePayment_DB_Failure(t *testing.T) {
	repo := mocks.NewPaymentRepository(t)
	service := NewPaymentService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	orderID := "order-001"
	dbErr := errors.New("db failure")

	repo.On("GetByOrderID", mock.Anything, orderID).
		Return(nil, dbErr)

	_, err := service.CreatePayment(
		context.Background(),
		orderID,
		1000,
		"USD",
	)
	require.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t,
		"CreatePayment",
		mock.Anything,
		mock.Anything,
	)
}

func TestCreatePayment_RaceCondition(t *testing.T) {
	repo := mocks.NewPaymentRepository(t)
	service := NewPaymentService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	orderID := "order-001"
	existingPayment := &domain.Payment{
		ID:          "payment-001",
		OrderID:     orderID,
		Currency:    "USD",
		AmountMinor: 1000,
	}

	repo.On("GetByOrderID", mock.Anything, orderID).
		Return(nil, domain.ErrPaymentNotFound).Once()

	repo.On("GetByOrderID", mock.Anything, orderID).
		Return(existingPayment, nil).Once()

	repo.On("CreatePayment", mock.Anything, mock.MatchedBy(func(payment *domain.Payment) bool {
		return payment.OrderID == orderID &&
			payment.Currency == "USD" &&
			payment.AmountMinor == 1000 &&
			payment.ID != ""
	})).Return(domain.ErrPaymentAlreadyExists).Once()

	got, err := service.CreatePayment(
		context.Background(),
		orderID,
		1000,
		"USD",
	)

	require.NoError(t, err)
	require.Equal(t, existingPayment, got)
}

func TestCreatePayment_RaceCondition_Failure(t *testing.T) {
	repo := mocks.NewPaymentRepository(t)
	service := NewPaymentService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	dbErr := errors.New("db failure")
	orderID := "order-001"
	existingPayment := &domain.Payment{
		OrderID:     orderID,
		Currency:    "USD",
		AmountMinor: 1000,
	}

	repo.On("GetByOrderID", mock.Anything, orderID).
		Return(nil, domain.ErrPaymentNotFound).Once()
	repo.On("GetByOrderID", mock.Anything, orderID).
		Return(nil, dbErr).Once()
	repo.On("CreatePayment", mock.Anything, mock.MatchedBy(func(payment *domain.Payment) bool {
		return payment.OrderID == existingPayment.OrderID &&
			payment.Currency == existingPayment.Currency &&
			payment.AmountMinor == existingPayment.AmountMinor &&
			payment.ID != ""
	})).Return(domain.ErrPaymentAlreadyExists).Once()

	_, err := service.CreatePayment(
		context.Background(),
		orderID,
		1000,
		"USD",
	)
	require.ErrorIs(t, err, dbErr)
}

func TestCreatePayment_CreateRepositoryFailure(t *testing.T) {
	repo := mocks.NewPaymentRepository(t)
	service := NewPaymentService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	orderID := "order-001"
	dbErr := errors.New("db failure")

	repo.On("GetByOrderID", mock.Anything, orderID).
		Return(nil, domain.ErrPaymentNotFound).Once()
	repo.On("CreatePayment", mock.Anything, mock.MatchedBy(func(payment *domain.Payment) bool {
		return payment.OrderID == orderID &&
			payment.Currency == "USD" &&
			payment.AmountMinor == 1000 &&
			payment.ID != ""
	})).Return(dbErr)

	_, err := service.CreatePayment(
		context.Background(),
		orderID,
		1000,
		"USD",
	)
	require.ErrorIs(t, err, dbErr)
}
