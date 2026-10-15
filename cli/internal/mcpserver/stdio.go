package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// stdioDrainTimeout bounds how long stdin EOF waits for outstanding JSON-RPC
// responses before the server side of the transport is closed. Without this
// grace period the SDK cancels in-flight handlers the moment EOF arrives, so
// piped input (data + immediate EOF, e.g. `echo ... | fabrik mcp`) is dropped
// and exits 0 with 0 bytes.
const stdioDrainTimeout = 15 * time.Second

// runStdio serves one MCP session over line-delimited JSON-RPC frames.
//
// Frames are read from stdin and forwarded to the SDK server through an
// internal pipe; server responses are forwarded to stdout. Request IDs seen
// on stdin are tracked, and stdin EOF is only propagated to the server after
// every outstanding request has received its response (or the drain timeout
// fires). This makes piped and interactive use deterministic: no response is
// ever lost to an EOF race, and EOF always terminates the process.
func runStdio(ctx context.Context, srv *mcp.Server, stdin io.Reader, stdout io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	toServerR, toServerW := io.Pipe()
	fromServerR, fromServerW := io.Pipe()
	transport := &mcp.IOTransport{Reader: toServerR, Writer: fromServerW}

	ss, err := srv.Connect(ctx, transport, nil)
	if err != nil {
		_ = toServerR.Close()
		_ = fromServerW.Close()
		return err
	}

	tracker := newPendingTracker()

	sessDone := make(chan error, 1)
	go func() { sessDone <- ss.Wait() }()

	var pumps sync.WaitGroup
	pumps.Add(2)
	stdinDone := make(chan struct{})
	go func() {
		defer pumps.Done()
		defer close(stdinDone)
		pumpStdin(stdin, toServerW, tracker)
	}()
	go func() {
		defer pumps.Done()
		pumpStdout(fromServerR, stdout, tracker)
	}()

	// Wait for stdin EOF, then let outstanding responses drain before
	// closing the server side of the transport (which ends the session).
	// Aborts early if the session or context ends first.
	select {
	case <-stdinDone:
	case <-sessDone:
	case <-ctx.Done():
	}
	select {
	case <-tracker.drained():
	case <-sessDone:
	case <-time.After(stdioDrainTimeout):
	case <-ctx.Done():
	}
	_ = toServerW.Close()
	_ = fromServerW.Close()

	pumps.Wait()
	select {
	case err := <-sessDone:
		return err
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// pumpStdin forwards stdin frames to the server, tracking request IDs.
// It returns on stdin EOF or an unrecoverable read/write error.
func pumpStdin(stdin io.Reader, toServer *io.PipeWriter, tracker *pendingTracker) {
	br := bufio.NewReader(stdin)
	for {
		line, err := readFrame(br)
		if len(line) > 0 {
			for _, id := range frameRequests(line) {
				tracker.add(id)
			}
			if _, werr := toServer.Write(ensureNewline(line)); werr != nil {
				tracker.setEOF()
				return
			}
		}
		if err != nil {
			tracker.setEOF()
			return
		}
	}
}

// pumpStdout forwards server frames to stdout, retiring request IDs as
// responses arrive. It returns when the server closes its side.
func pumpStdout(fromServer io.Reader, stdout io.Writer, tracker *pendingTracker) {
	br := bufio.NewReader(fromServer)
	for {
		line, err := readFrame(br)
		if len(line) > 0 {
			if _, werr := stdout.Write(ensureNewline(line)); werr != nil {
				return
			}
			for _, id := range frameResponses(line) {
				tracker.remove(id)
			}
		}
		if err != nil {
			return
		}
	}
}

// readFrame reads one newline-terminated frame, tolerating a final line
// without a trailing newline and CRLF line endings. It returns
// (nil, io.EOF) once the stream is exhausted.
func readFrame(br *bufio.Reader) ([]byte, error) {
	line, err := br.ReadBytes('\n')
	line = bytes.TrimSuffix(line, []byte{'\n'})
	line = bytes.TrimSuffix(line, []byte{'\r'})
	line = bytes.TrimSpace(line)
	if len(line) == 0 && err != nil {
		return nil, err
	}
	return line, err
}

func ensureNewline(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		return line
	}
	return append(append([]byte(nil), line...), '\n')
}

// frameRequests returns the normalized IDs of JSON-RPC requests (which carry
// both "id" and "method") found in a raw line. A line may hold several
// whitespace-separated values; all are scanned.
func frameRequests(line []byte) []string {
	var out []string
	eachValue(line, func(v map[string]any) {
		id, ok := v["id"]
		if !ok || id == nil {
			return
		}
		if _, ok := v["method"]; !ok {
			return
		}
		out = append(out, idKey(id))
	})
	return out
}

// frameResponses returns the normalized IDs of JSON-RPC responses (which
// carry "id" plus "result" or "error") found in a raw line.
func frameResponses(line []byte) []string {
	var out []string
	eachValue(line, func(v map[string]any) {
		id, ok := v["id"]
		if !ok || id == nil {
			return
		}
		if _, ok := v["result"]; !ok {
			if _, ok := v["error"]; !ok {
				return
			}
		}
		out = append(out, idKey(id))
	})
	return out
}

func eachValue(line []byte, fn func(map[string]any)) {
	dec := json.NewDecoder(bytes.NewReader(line))
	for {
		var v map[string]any
		if err := dec.Decode(&v); err != nil {
			return
		}
		fn(v)
	}
}

// idKey normalizes a JSON-RPC id (number or string) for map comparison.
func idKey(id any) string {
	return fmt.Sprintf("%T:%v", id, id)
}

// pendingTracker records outstanding request IDs. drained closes once stdin
// has hit EOF and no IDs remain.
type pendingTracker struct {
	mu      sync.Mutex
	pending map[string]bool
	eof     bool
	ch      chan struct{}
	once    sync.Once
}

func newPendingTracker() *pendingTracker {
	return &pendingTracker{pending: map[string]bool{}, ch: make(chan struct{})}
}

func (t *pendingTracker) add(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pending[id] = true
}

func (t *pendingTracker) remove(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.pending, id)
	t.checkLocked()
}

func (t *pendingTracker) setEOF() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.eof = true
	t.checkLocked()
}

func (t *pendingTracker) checkLocked() {
	if t.eof && len(t.pending) == 0 {
		t.once.Do(func() { close(t.ch) })
	}
}

func (t *pendingTracker) drained() <-chan struct{} { return t.ch }
