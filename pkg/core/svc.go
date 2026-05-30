// Package core provides core service logic and interfaces.
package core

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// userRepo defines the interface for user repository operations.
type userRepo interface {
	CheckHealth(ctx context.Context) error
}

// Service encapsulates core business logic and dependencies.
type Service struct {
	users userRepo
}

// New creates a new Service instance with the provided userRepo and someAPI.
func New(users userRepo) *Service {
	return &Service{
		users: users,
	}
}

// CheckHealth checks the health of the core service and its dependencies.
func (s *Service) CheckHealth(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error { return s.users.CheckHealth(ctx) })

	return eg.Wait()
}
