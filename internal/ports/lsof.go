package ports

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// LsofProvider shells out to lsof to discover active ports.
type LsofProvider struct {
	// Path to the lsof executable. Defaults to "lsof" when empty.
	Path string
}

// NewSystemProvider returns a Provider that uses lsof, the most
// widely-available cross-platform utility for enumerating open ports.
func NewSystemProvider() Provider {
	return &LsofProvider{}
}

// List executes lsof and converts the results into Port entries.
func (p *LsofProvider) List(ctx context.Context) ([]Port, error) {
	path := p.Path
	if path == "" {
		path = "lsof"
	}

	args := []string{"-nP", "-iTCP", "-sTCP:LISTEN", "-iUDP", "-FpcfLnuPT"}
	cmd := exec.CommandContext(ctx, path, args...)
	output, err := cmd.Output()
	if err != nil {
		if ee := (&exec.ExitError{}); errors.As(err, &ee) {
			return nil, fmt.Errorf("lsof failed: %w", err)
		}
		return nil, fmt.Errorf("executing %s: %w", path, err)
	}

	entries, err := parseLsofOutput(string(output))
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Port == entries[j].Port {
			if entries[i].Protocol == entries[j].Protocol {
				if entries[i].PID == entries[j].PID {
					return entries[i].Address < entries[j].Address
				}
				return entries[i].PID < entries[j].PID
			}
			return entries[i].Protocol < entries[j].Protocol
		}
		return entries[i].Port < entries[j].Port
	})

	return entries, nil
}

func parseLsofOutput(out string) ([]Port, error) {
	scanner := bufio.NewScanner(strings.NewReader(out))

	var (
		entries      []Port
		entry        *Port
		proc         processContext
		dedupe       = make(map[string]struct{})
		flushCurrent = func() {
			if entry == nil {
				return
			}
			if entry.Port == 0 {
				entry = nil
				return
			}
			if entry.Process == "" {
				entry.Process = proc.command
			}
			if entry.User == "" {
				entry.User = proc.user
			}
			key := fmt.Sprintf("%d|%s|%d|%s", entry.PID, entry.Protocol, entry.Port, entry.Address)
			if _, seen := dedupe[key]; seen {
				entry = nil
				return
			}
			dedupe[key] = struct{}{}
			entries = append(entries, *entry)
			entry = nil
		}
	)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		field := line[0]
		value := line[1:]

		switch field {
		case 'p':
			flushCurrent()
			pid, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("parse pid %q: %w", value, err)
			}
			proc = processContext{pid: pid}
		case 'c':
			proc.command = value
		case 'L':
			proc.user = value
		case 'u':
			if proc.user == "" {
				proc.user = value
			}
		case 'f':
			flushCurrent()
			entry = &Port{PID: proc.pid, Process: proc.command, User: proc.user}
		case 'P':
			if entry != nil {
				entry.Protocol = strings.ToLower(value)
			}
		case 'n':
			if entry != nil {
				host, port := splitHostPort(value)
				entry.Address = host
				if portInt, err := strconv.Atoi(port); err == nil {
					entry.Port = portInt
				}
			}
		case 'T':
			if entry != nil {
				if strings.HasPrefix(value, "ST=") {
					entry.State = strings.TrimPrefix(value, "ST=")
				}
			}
		}
	}

	flushCurrent()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan lsof output: %w", err)
	}

	return entries, nil
}

type processContext struct {
	pid     int
	command string
	user    string
}

func splitHostPort(addr string) (string, string) {
	if addr == "" {
		return "", ""
	}

	if strings.HasPrefix(addr, "[") {
		if idx := strings.LastIndex(addr, "]:"); idx != -1 {
			return strings.TrimPrefix(addr[:idx], "["), addr[idx+2:]
		}
		return strings.Trim(addr, "[]"), ""
	}

	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		host := addr[:idx]
		port := addr[idx+1:]
		if host == "" {
			host = "*"
		}
		return host, port
	}

	return addr, ""
}
