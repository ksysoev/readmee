package cmd

import (
	"context"
	"fmt"

	"github.com/ksysoev/readmee/pkg/core"
	"github.com/ksysoev/readmee/pkg/repo/user"
	"github.com/ksysoev/readmee/pkg/ssh"
	"github.com/redis/go-redis/v9"
)

// RunCommand initializes the logger, loads configuration, creates the core and API services,
// and starts the API service. It returns an error if any step fails.
func RunCommand(ctx context.Context, flags *cmdFlags) error {
	if err := initLogger(flags); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	cfg, err := loadConfig(flags)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})

	userRepo := user.New(rdb)
	svc := core.New(userRepo)

	sshSvc, err := ssh.New(cfg.SSH, svc)
	if err != nil {
		return fmt.Errorf("failed to create ssh service: %w", err)
	}

	err = sshSvc.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run ssh service: %w", err)
	}

	return nil
}
