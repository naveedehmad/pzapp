package ports

import (
	"fmt"
	"testing"
)

func TestParseLsofOutput(t *testing.T) {
	raw := `p1234
cnpm
Lnaveed
f11
PTCP
n*:3000
TST=LISTEN
f12
PTCP
n127.0.0.1:9229
TST=LISTEN
p2222
cnpm
Lnaveed
f10
PTCP
n*:3000
TST=LISTEN
f10
PTCP
n*:3000
TST=LISTEN
p3333
cnode
Lnaveed
f5
PUDP
n*:68
`

	entries, err := parseLsofOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	want := map[string]Port{
		"1234/3000": {PID: 1234, Process: "npm", User: "naveed", Protocol: "tcp", Port: 3000, Address: "*", State: "LISTEN"},
		"1234/9229": {PID: 1234, Process: "npm", User: "naveed", Protocol: "tcp", Port: 9229, Address: "127.0.0.1", State: "LISTEN"},
		"2222/3000": {PID: 2222, Process: "npm", User: "naveed", Protocol: "tcp", Port: 3000, Address: "*", State: "LISTEN"},
		"3333/68":   {PID: 3333, Process: "node", User: "naveed", Protocol: "udp", Port: 68, Address: "*", State: ""},
	}

	for _, entry := range entries {
		key := keyFor(entry.PID, entry.Port)
		expected, ok := want[key]
		if !ok {
			t.Fatalf("unexpected entry: %+v", entry)
		}
		if entry.Process != expected.Process || entry.Protocol != expected.Protocol || entry.Address != expected.Address || entry.State != expected.State {
			t.Fatalf("mismatch for %s: got %+v want %+v", key, entry, expected)
		}
		delete(want, key)
	}

	if len(want) != 0 {
		t.Fatalf("missing expected entries: %+v", want)
	}
}

func keyFor(pid, port int) string {
	return fmt.Sprintf("%d/%d", pid, port)
}
