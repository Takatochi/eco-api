package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"eco-api/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type serviceMock struct {
	createFn func(ctx context.Context, in model.MeasurementIn) (int64, error)
	listFn   func(ctx context.Context, deviceID, fromStr, toStr, limitStr string) ([]model.MeasurementOut, error)
}

type dbPingerMock struct {
	pingFn func(ctx context.Context) error
}

func (m *dbPingerMock) Ping(ctx context.Context) error {
	if m.pingFn == nil {
		return nil
	}
	return m.pingFn(ctx)
}

func (m *serviceMock) Create(ctx context.Context, in model.MeasurementIn) (int64, error) {
	if m.createFn == nil {
		return 0, errors.New("create not implemented")
	}
	return m.createFn(ctx, in)
}

func (m *serviceMock) List(ctx context.Context, deviceID, fromStr, toStr, limitStr string) ([]model.MeasurementOut, error) {
	if m.listFn == nil {
		return nil, errors.New("list not implemented")
	}
	return m.listFn(ctx, deviceID, fromStr, toStr, limitStr)
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ping", h.Ping)
	r.GET("/health", h.Health)
	r.GET("/openapi.json", h.OpenAPI)
	r.POST("/api/measurements", h.CreateMeasurement)
	r.GET("/api/measurements", h.ListMeasurements)
	return r
}

func TestPing(t *testing.T) {
	h := New(&serviceMock{}, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !bytes.Contains(res.Body.Bytes(), []byte(`"pong"`)) {
		t.Fatalf("expected pong payload, got %s", res.Body.String())
	}
}

func TestOpenAPI(t *testing.T) {
	h := New(&serviceMock{}, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !bytes.Contains(res.Body.Bytes(), []byte(`"openapi"`)) {
		t.Fatalf("expected openapi document, got %s", res.Body.String())
	}
}

func TestCreateMeasurementInvalidJSON(t *testing.T) {
	h := New(&serviceMock{}, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/measurements", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCreateMeasurementConflict(t *testing.T) {
	h := New(&serviceMock{
		createFn: func(ctx context.Context, in model.MeasurementIn) (int64, error) {
			return 0, &pgconn.PgError{Code: "23505", Message: "duplicate key"}
		},
	}, nil)
	r := setupRouter(h)

	body := map[string]any{"deviceId": "sensor-1", "timestamp": "2026-02-10T10:00:00Z"}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/measurements", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.Code)
	}
}

func TestListMeasurements(t *testing.T) {
	h := New(&serviceMock{
		listFn: func(ctx context.Context, deviceID, fromStr, toStr, limitStr string) ([]model.MeasurementOut, error) {
			if deviceID == "" {
				return nil, errors.New("required query params: deviceId, from, to (RFC3339)")
			}
			return []model.MeasurementOut{{ID: 1, DeviceID: deviceID, Timestamp: "2026-02-10T10:00:00Z"}}, nil
		},
	}, nil)
	r := setupRouter(h)

	reqBad := httptest.NewRequest(http.MethodGet, "/api/measurements", nil)
	resBad := httptest.NewRecorder()
	r.ServeHTTP(resBad, reqBad)
	if resBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid query, got %d", resBad.Code)
	}

	reqOK := httptest.NewRequest(http.MethodGet, "/api/measurements?deviceId=sensor-1&from=2026-02-10T09:00:00Z&to=2026-02-10T11:00:00Z", nil)
	resOK := httptest.NewRecorder()
	r.ServeHTTP(resOK, reqOK)
	if resOK.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resOK.Code)
	}
	if !bytes.Contains(resOK.Body.Bytes(), []byte(`"deviceId":"sensor-1"`)) {
		t.Fatalf("expected payload with sensor-1, got %s", resOK.Body.String())
	}
}

func TestHealth(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		h := New(&serviceMock{}, &dbPingerMock{
			pingFn: func(ctx context.Context) error { return nil },
		})
		r := setupRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.Code)
		}
		if !bytes.Contains(res.Body.Bytes(), []byte(`"ok"`)) {
			t.Fatalf("expected status ok, got %s", res.Body.String())
		}
	})

	t.Run("db not ready", func(t *testing.T) {
		h := New(&serviceMock{}, &dbPingerMock{
			pingFn: func(ctx context.Context) error { return errors.New("db down") },
		})
		r := setupRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", res.Code)
		}
	})
}
