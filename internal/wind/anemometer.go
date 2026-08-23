// Package wind models tower-crane anemometer faults with cross-package Is.
package wind

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrWindGust      = errors.New("wind: gust spike")
	ErrSustainedHigh = errors.New("wind: sustained high")
	ErrSensorStale   = errors.New("wind: sensor stale")
)

type Fault struct {
	RigID string
	Cause error
	At    time.Time
}

func (f *Fault) Error() string { return fmt.Sprintf("wind rig=%s: %v", f.RigID, f.Cause) }
func (f *Fault) Unwrap() error { return f.Cause }

func Wrap(rigID string, cause error) error {
	if cause == nil {
		return nil
	}
	return &Fault{RigID: rigID, Cause: cause, At: time.Now().UTC()}
}

func IsRecoverable(err error) bool { return errors.Is(err, ErrWindGust) }
func IsBan(err error) bool         { return errors.Is(err, ErrSustainedHigh) }

func Classify(err error) string {
	switch {
	case errors.Is(err, ErrWindGust):
		return "gust"
	case errors.Is(err, ErrSustainedHigh):
		return "ban"
	default:
		return "unknown"
	}
}
