// Package journal writes operator event journal lines.
package journal

import (
	"fmt"
	"os"
	"sync"
)

type Writer struct {
	mu      sync.Mutex
	path    string
	buf     []string
	maxBuf  int
	enabled bool
}

func NewWriter(path string, maxBuf int) *Writer {
	if maxBuf <= 0 {
		maxBuf = 1000
	}
	return &Writer{path: path, buf: []string{}, maxBuf: maxBuf, enabled: path != ""}
}

func (w *Writer) Append(tick int64, rigID, kind, detail string) {
	line := fmt.Sprintf("tick=%d rig=%s kind=%s detail=%s", tick, rigID, kind, detail)
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) >= w.maxBuf {
		w.buf = w.buf[1:]
	}
	w.buf = append(w.buf, line)
	if w.enabled {
		f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = fmt.Fprintln(f, line)
			_ = f.Close()
		}
	}
}

func (w *Writer) Snapshot() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, len(w.buf))
	copy(out, w.buf)
	return out
}
