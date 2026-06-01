// Package api provides the implementation of the API server for the application.
package ssh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/bubbletea"
	"github.com/charmbracelet/ssh"
)

type SSH struct {
	svc    Service
	config Config
	wg     sync.WaitGroup
	server *ssh.Server
}

type Config struct {
	Listen     string `mapstructure:"listen"`
	PrivateKey string `mapstructure:"private_key"`
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

	if svc == nil {
		return nil, fmt.Errorf("service must be provided")
	}

	s := &SSH{
		config: cfg,
		svc:    svc,
	}

	server, err := wish.NewServer(
		wish.WithAddress(cfg.Listen),
		wish.WithHostKeyPEM([]byte(cfg.PrivateKey)),
		wish.WithPublicKeyAuth(func(_ ssh.Context, _ ssh.PublicKey) bool {
			return true
		}),
		wish.WithMiddleware(
			bubbletea.Middleware(s.appRouter), // Bubble Tea apps usually require an app router.
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH server: %w", err)
	}

	s.server = server

	return s, nil
}

// Run starts the API server with the provided configuration.
// It listens on the address specified in the configuration and handles graceful shutdown.
// The server will log any errors encountered during shutdown.
// If the server fails to start, it returns an error.
func (s *SSH) Run(ctx context.Context) error {
	defer s.wg.Wait()

	s.wg.Go(func() {
		<-ctx.Done()
		slog.DebugContext(ctx, "shutting down SSH server")

		if err := s.server.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			slog.Error("Could not stop server", "error", err)
		}
	})

	return s.server.ListenAndServe()
}

func (s *SSH) appRouter(session ssh.Session) (tea.Model, []tea.ProgramOption) {
	slog.Info("New SSH session", "user", session.User(), "remote_addr", session.RemoteAddr(), "command", session.Command())
	_, _, active := session.Pty()
	if !active {
		slog.Warn("No PTY requested, closing session")
		// return nil, nil
	}

	args := session.Command()

	var appName string
	if len(args) > 0 {
		appName = args[0]
		slog.Info("Running app", "app_name", appName)
	}

	opts := []tea.ProgramOption{tea.MakeOptions(sess), tea.WithAltScreen()}

	return model(5), opts
}

type model int

// Init optionally returns an initial command we should run. In this case we
// want to start the timer.
func (m model) Init() tea.Cmd {
	return tick
}

// Update is called when messages are received. The idea is that you inspect the
// message and send back an updated model accordingly. You can also return
// a command, which is a function that performs I/O and returns a message.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+z":
			return m, tea.Suspend
		}

	case tickMsg:
		m--
		if m <= 0 {
			return m, tea.Quit
		}
		return m, tick
	}
	return m, nil
}

// View returns a string based on data in the model. That string which will be
// rendered to the terminal.
func (m model) View() tea.View {
	return tea.NewView(fmt.Sprintf("Hi. This program will exit in %d seconds.\n\nTo quit sooner press ctrl-c, or press ctrl-z to suspend...\n", m))
}

// Messages are events that we respond to in our Update function. This
// particular one indicates that the timer has ticked.
type tickMsg time.Time

func tick() tea.Msg {
	time.Sleep(time.Second)
	return tickMsg{}
}
