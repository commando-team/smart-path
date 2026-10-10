package service

import "context"

type PaymentService struct{}

func NewPaymentService() *PaymentService {
	return &PaymentService{}
}

func (s *PaymentService) Pay(ctx context.Context) error {
	return nil
}
