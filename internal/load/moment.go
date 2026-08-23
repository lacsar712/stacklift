// Package load models hook load and moment-percent limiting.
package load

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrMomentExceeded = errors.New("load: moment exceeded")
	ErrStaleLoad      = errors.New("load: stale sample")
	ErrUnderLoad      = errors.New("load: under minimum")
)

type Fault struct {
	RigID string
	Cause error
	At    time.Time
}

func (f *Fault) Error() string {
	if f == nil {
		return "load: nil fault"
	}
	return fmt.Sprintf("load fault rig=%s: %v", f.RigID, f.Cause)
}
func (f *Fault) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Cause
}

func Wrap(rigID string, cause error) error {
	if cause == nil {
		return nil
	}
	return &Fault{RigID: rigID, Cause: cause, At: time.Now().UTC()}
}

func IsMoment(err error) bool { return errors.Is(err, ErrMomentExceeded) }
func IsStale(err error) bool  { return errors.Is(err, ErrStaleLoad) }

func Classify(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrMomentExceeded):
		return "moment"
	case errors.Is(err, ErrStaleLoad):
		return "stale"
	default:
		return "unknown"
	}
}
