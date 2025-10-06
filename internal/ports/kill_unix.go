//go:build unix

package ports

import (
	"fmt"
	"syscall"
	"time"
)

// Terminate attempts to gracefully kill a process, escalating from SIGTERM to SIGKILL if necessary.
func Terminate(pid int) error {
	// First, try SIGTERM (graceful termination)
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to PID %d: %w", pid, err)
	}

	// Wait a moment to see if process terminates gracefully
	time.Sleep(500 * time.Millisecond)

	// Check if process still exists
	if err := syscall.Kill(pid, 0); err != nil {
		// Process is gone (syscall.Kill with signal 0 just checks existence)
		return nil // Success!
	}

	// Process is still alive, escalate to SIGKILL (forceful termination)
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to send SIGKILL to PID %d: %w", pid, err)
	}

	// Wait a bit more and check again
	time.Sleep(200 * time.Millisecond)
	if err := syscall.Kill(pid, 0); err != nil {
		return nil // Process finally died
	}

	return fmt.Errorf("process %d survived both SIGTERM and SIGKILL - it's unstoppable! 💀", pid)
}
