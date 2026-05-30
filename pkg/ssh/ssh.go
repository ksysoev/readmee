// Package api provides the implementation of the API server for the application.
package ssh

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"log/slog"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type SSH struct {
	svc       Service
	config    Config
	sshConfig *ssh.ServerConfig
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

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		<-ctx.Done()
		slog.DebugContext(ctx, "shutting down SSH server")

		listener.Close()
	}()

	return s.serve(ctx, listener)
}

func (s *SSH) serve(ctx context.Context, listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		s.wg.Add(1)

		go func() {
			defer s.wg.Done()

			s.handleConn(ctx, conn)
		}()
	}
}

func (s *SSH) handleConn(ctx context.Context, nConn net.Conn) {
	defer nConn.Close()

	conn, chans, reqs, err := ssh.NewServerConn(nConn, s.sshConfig)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	// The incoming Request channel must be serviced.
	wg.Add(1)

	go func() {
		ssh.DiscardRequests(reqs)
		wg.Done()
	}()

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of a shell, the type is
		// "session" and ServerShell may be used to present a simple
		// terminal interface.
		if newChannel.ChannelType() != "session" {
			err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			if err != nil {
				log.Printf("Could not reject channel: %v", err)
			}

			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
		}

		// Sessions have out-of-band requests such as "shell",
		// "pty-req" and "env".  Here we handle only the
		// "shell" request.
		wg.Add(1)

		go func(in <-chan *ssh.Request) {
			for req := range in {
				err := req.Reply(req.Type == "shell", nil)
				if err != nil {
					log.Printf("Failed to reply to request: %v", err)
				}
			}

			wg.Done()
		}(requests)

		terminal := term.NewTerminal(channel, "> ")

		wg.Add(1)

		go func() {
			defer func() {
				channel.Close()
				wg.Done()
			}()

			for {
				line, err := terminal.ReadLine()
				if err != nil {
					break
				}

				fmt.Println(line)
			}
		}()
	}
}
