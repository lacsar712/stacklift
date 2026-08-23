package web_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/web"
)

func TestEmbeddedAssets(t *testing.T) {
	for _, name := range []string{"index.html", "style.css", "app.js"} {
		f, err := web.Open(name)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		_ = f.Close()
	}
}

func TestHandler(t *testing.T) {
	h := web.Handler()
	if h == nil {
		t.Fatal("handler")
	}
}
