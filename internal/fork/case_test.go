package fork

import (
	"context"
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/model"
)

func TestCase(t *testing.T) {
	tel := NewTelescope("CR-01", 1000)
	tel.left.extensionMM = 0
	tel.right.extensionMM = 200
	err := tel.CheckLoadBalance(50)
	if err == nil {
		t.Fatal("expected load imbalance error")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("load imbalance should not be DeadlineExceeded")
	}
}
