package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lacsar712/stacklift/internal/app"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestHealth(t *testing.T) {
	svc, _ := app.New(config.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	svc.Server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
}

func TestSlewAPI(t *testing.T) {
	svc, _ := app.New(config.Default())
	svc.IngestLoad(model.LoadSample{RigID: "TC-01", MomentPct: 50, At: time.Now()})
	body, _ := json.Marshal(map[string]any{"rig_id": "TC-01", "target_az": 10})
	req := httptest.NewRequest(http.MethodPost, "/api/slew", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	svc.Server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
