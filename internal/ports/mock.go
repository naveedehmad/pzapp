package ports

import (
	"context"
	"time"
)

// MockProvider returns a deterministic list of ports for early UI development.
type MockProvider struct{}

// NewMockProvider constructs a Provider that serves mock data.
func NewMockProvider() Provider {
	return &MockProvider{}
}

// List returns a fixed slice of synthetic port entries.
func (MockProvider) List(ctx context.Context) ([]Port, error) {
	sample := []Port{
		{PID: 4521, Process: "node", User: "naveed", Protocol: "tcp", Port: 3000, Address: "0.0.0.0", State: "LISTEN"},
		{PID: 9112, Process: "postgres", User: "postgres", Protocol: "tcp", Port: 5432, Address: "127.0.0.1", State: "LISTEN"},
		{PID: 2048, Process: "redis-server", User: "redis", Protocol: "tcp", Port: 6379, Address: "127.0.0.1", State: "LISTEN"},
		{PID: 7320, Process: "python", User: "naveed", Protocol: "tcp", Port: 8000, Address: "127.0.0.1", State: "LISTEN"},
		{PID: 8871, Process: "nginx", User: "root", Protocol: "tcp", Port: 443, Address: "0.0.0.0", State: "LISTEN"},
	}

	// Simulate a tiny delay to exercise the loading state.
	select {
	case <-time.After(120 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return sample, nil
}
