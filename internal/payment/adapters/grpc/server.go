package grpc

import (
	"context"
	"errors"
	"grpc-course/gen/payment"
	"grpc-course/internal/payment/application"
	"grpc-course/internal/payment/domain"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	payment.UnimplementedPaymentServiceServer
	service paymentCreator
}

type paymentCreator interface {
	CreatePayment(ctx context.Context, orderID string, amountMinor int64, currency string) (*domain.Payment, error)
}

func NewServer(service *application.PaymentService) *Server {
	return newServer(service)
}

func newServer(service paymentCreator) *Server {
	return &Server{
		service: service,
	}
}

func (s *Server) CreatePayment(
	ctx context.Context,
	req *payment.CreatePaymentRequest,
) (*payment.CreatePaymentResponse, error) {
	log.Println("Payment CreatePayment called")

	pay, err := s.service.CreatePayment(ctx, req.GetOrderId(), req.GetAmountMinor(), req.GetCurrency())
	if errors.Is(err, domain.ErrPaymentOrderIDRequired) {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}
	if errors.Is(err, domain.ErrAmountMinorNegative) {
		return nil, status.Error(codes.InvalidArgument, "amount minor must not be negative")
	}
	if errors.Is(err, domain.ErrPaymentCurrencyRequired) {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "create payment error")
	}

	response := &payment.CreatePaymentResponse{
		PaymentId: pay.ID,
	}

	return response, nil
}
