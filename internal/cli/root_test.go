package cli

import (
	"context"
	"testing"

	"github.com/pauloborges/orb-cli/internal/mcpserver"
)

func TestRootCommandIncludesMCP(t *testing.T) {
	root := NewRootCmd()
	for _, command := range root.Commands() {
		if command.Name() == "mcp" {
			return
		}
	}
	t.Fatal("root command does not include mcp")
}

func TestMCPCommandPassesCommandContext(t *testing.T) {
	t.Setenv(mcpserver.LiveAPIKeyEnv, "live-key")
	t.Setenv(mcpserver.TestAPIKeyEnv, "test-key")

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "from-main")
	var received context.Context
	command := newMCPCmd(func(got context.Context, config mcpserver.Config) error {
		received = got
		if config.LiveAPIKey != "live-key" || config.TestAPIKey != "test-key" {
			t.Fatalf("unexpected MCP config: %#v", config)
		}
		return nil
	})
	command.SetContext(ctx)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := received.Value(contextKey{}); got != "from-main" {
		t.Fatalf("RunE context value = %v, want from-main", got)
	}
}
