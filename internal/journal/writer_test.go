package journal_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/journal"
)

func TestWriter(t *testing.T) {
	w := journal.NewWriter("", 5)
	w.Append(1, "TC-01", "k", "d")
	if len(w.Snapshot()) != 1 {
		t.Fatal("line")
	}
}
