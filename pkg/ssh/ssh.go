// Package api provides the implementation of the API server for the application.
package ssh

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

type SSH struct {
	svc       Service
	sshConfig *ssh.ServerConfig
	config    Config
	wg        sync.WaitGroup
}

type Config struct {
	Listen string `mapstructure:"listen"`
}

type Service interface {
	CheckHealth(ctx context.Context) error
}

// New creates a new API instance with the provided configuration and service.
// It validates the configuration and returns an error if the listen address is not specified.
func New(cfg Config, svc Service) (*SSH, error) {
	if cfg.Listen == "" {
		return nil, fmt.Errorf("listen address must be specified")
	}

	config := &ssh.ServerConfig{
		// Remove to disable public key auth.
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{
				// Record the public key used for authentication.
				Extensions: map[string]string{
					"pubkey-fp": ssh.FingerprintSHA256(pubKey),
				},
			}, nil
		},
	}

	privateKey, err := rsa.GenerateKey(nil, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %v", err)
	}

	config.AddHostKey(signer)

	api := &SSH{
		config:    cfg,
		svc:       svc,
		sshConfig: config,
	}

	return api, nil
}

// Run starts the API server with the provided configuration.
// It listens on the address specified in the configuration and handles graceful shutdown.
// The server will log any errors encountered during shutdown.
// If the server fails to start, it returns an error.
func (s *SSH) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.Listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", s.config.Listen, err)
	}

	slog.InfoContext(ctx, "SSH server is listening", "addr", s.config.Listen)

	defer s.wg.Wait()

	s.wg.Go(func() {
		<-ctx.Done()
		slog.DebugContext(ctx, "shutting down SSH server")

		listener.Close()
	})

	return s.serve(ctx, listener)
}

func (s *SSH) serve(ctx context.Context, listener net.Listener) error {
	for {
		conn, err := listener.Accept()

		switch {
		case errors.Is(err, net.ErrClosed):
			return nil
		case err != nil:
			return fmt.Errorf("failed to accept connection: %v", err)
		}

		s.wg.Go(func() {
			s.handleConn(ctx, conn)
		})
	}
}

func (s *SSH) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	sConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshConfig)
	if err != nil {
		return
	}

	slog.DebugContext(ctx, "new SSH connection", "remote", sConn.RemoteAddr())

	s.wg.Go(func() {
		ssh.DiscardRequests(reqs)
	})

	// Service the incoming Channel channel.
	for {
		select {
		case <-ctx.Done():
			return
		case newChannel, ok := <-chans:
			if !ok {
				return
			}

			if newChannel.ChannelType() != "session" {
				err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
				if err != nil {
					slog.ErrorContext(ctx, "failed to reject channel", "error", err)
				}

				continue
			}

			channel, requests, err := newChannel.Accept()
			if err != nil {
				slog.ErrorContext(ctx, "could not accept channel", "error", err)
			}

			s.wg.Go(func() {
				s.handleChannel(ctx, channel, requests)
			})
		}
	}
}

func (s *SSH) handleChannel(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()
	// Sessions have out-of-band requests such as "shell",
	// "pty-req" and "env".  Here we handle only the
	// "shell" request.
	terminal := term.NewTerminal(channel, "> ")

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		for {
			_, err := terminal.ReadLine()
			if err != nil {
				return fmt.Errorf("failed to read line: %v", err)
			}
		}
	})

	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return io.EOF
			case req, ok := <-requests:
				if !ok {
					return io.EOF
				}

				if err := req.Reply(req.Type == "shell", nil); err != nil {
					return fmt.Errorf("failed to reply to request: %v", err)
				}
			}
		}
	})

	orgErr := eg.Wait()
	if !errors.Is(orgErr, io.EOF) || !errors.Is(orgErr, context.Canceled) {
		slog.ErrorContext(ctx, "error handling channel", "error", orgErr)
	}
}
