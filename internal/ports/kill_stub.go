//go:build !unix

package ports

import "fmt"

// Terminate is not implemented on non-Unix systems yet.
func Terminate(pid int) error {
    return fmt.Errorf("process termination is not supported on this platform")
}
