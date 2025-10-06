package ports

import "context"

// Port captures a single network port owned by a process.
type Port struct {
	PID      int
	Process  string
	User     string
	Protocol string
	Port     int
	Address  string
	State    string
}

// Provider enumerates active network ports on the system.
type Provider interface {
	List(ctx context.Context) ([]Port, error)
}
