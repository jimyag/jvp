package service

import "context"

type Service struct {
}

func New() (*Service, error) {
	return &Service{}, nil
}

func (s *Service) Run(ctx context.Context) error {
	return nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	return nil
}
