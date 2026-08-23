package boom_test

import (
	"context"
	"testing"

	"github.com/lacsar712/stacklift/internal/boom"
)

func TestMove(t *testing.T) {
	g := boom.NewGeometry(5, 60, 15, 75)
	d := boom.NewDriver(g, 40, 5)
	d.Ensure("TC-01")
	_, err := d.Move(context.Background(), boom.MoveRequest{RigID: "TC-01", TargetRadiusM: 30})
	if err != nil {
		t.Fatal(err)
	}
}
