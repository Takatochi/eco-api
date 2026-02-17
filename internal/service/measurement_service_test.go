package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"eco-api/internal/model"
)

type repoMock struct {
	createFn    func(ctx context.Context, in model.MeasurementIn, ts time.Time, dataHash string) (int64, error)
	setAnchorFn func(ctx context.Context, id int64, txHash string, blockNumber uint64) error
	listFn      func(ctx context.Context, deviceID string, from, to time.Time, limit int) ([]model.MeasurementRecord, error)
}

func (m *repoMock) Create(ctx context.Context, in model.MeasurementIn, ts time.Time, dataHash string) (int64, error) {
	if m.createFn == nil {
		return 0, errors.New("create not implemented")
	}
	return m.createFn(ctx, in, ts, dataHash)
}

func (m *repoMock) SetAnchorInfo(ctx context.Context, id int64, txHash string, blockNumber uint64) error {
	if m.setAnchorFn == nil {
		return errors.New("set anchor not implemented")
	}
	return m.setAnchorFn(ctx, id, txHash, blockNumber)
}

func (m *repoMock) List(ctx context.Context, deviceID string, from, to time.Time, limit int) ([]model.MeasurementRecord, error) {
	if m.listFn == nil {
		return nil, errors.New("list not implemented")
	}
	return m.listFn(ctx, deviceID, from, to, limit)
}

type anchorMock struct {
	anchorFn func(ctx context.Context, dataHashHex string) (string, uint64, error)
}

func (m *anchorMock) Anchor(ctx context.Context, dataHashHex string) (string, uint64, error) {
	if m.anchorFn == nil {
		return "", 0, errors.New("anchor not implemented")
	}
	return m.anchorFn(ctx, dataHashHex)
}

func TestCreateValidation(t *testing.T) {
	svc := NewMeasurementService(&repoMock{}, nil)

	_, err := svc.Create(context.Background(), model.MeasurementIn{})
	if err == nil || err.Error() != "deviceId is required" {
		t.Fatalf("expected deviceId validation error, got %v", err)
	}
}

func TestCreatePassesParsedTimestampAndHashToRepo(t *testing.T) {
	const tsString = "2026-02-10T10:00:00Z"
	input := model.MeasurementIn{DeviceID: "sensor-1", Timestamp: tsString}

	called := false
	svc := NewMeasurementService(&repoMock{
		createFn: func(ctx context.Context, in model.MeasurementIn, ts time.Time, dataHash string) (int64, error) {
			called = true
			if in.DeviceID != "sensor-1" {
				t.Fatalf("unexpected device id: %s", in.DeviceID)
			}
			if ts.UTC().Format(time.RFC3339) != tsString {
				t.Fatalf("unexpected parsed timestamp: %s", ts.UTC().Format(time.RFC3339))
			}
			if len(dataHash) != 66 || dataHash[:2] != "0x" {
				t.Fatalf("unexpected hash format: %s", dataHash)
			}
			return 42, nil
		},
	}, nil)

	id, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected repo.Create to be called")
	}
	if id != 42 {
		t.Fatalf("expected id 42, got %d", id)
	}
}

func TestCreateAnchorSuccessPersistsTxInfo(t *testing.T) {
	const tsString = "2026-02-10T10:00:00Z"
	const id int64 = 99
	input := model.MeasurementIn{DeviceID: "sensor-1", Timestamp: tsString}

	setAnchorCalled := false
	svc := NewMeasurementService(&repoMock{
		createFn: func(ctx context.Context, in model.MeasurementIn, ts time.Time, dataHash string) (int64, error) {
			return id, nil
		},
		setAnchorFn: func(ctx context.Context, gotID int64, txHash string, blockNumber uint64) error {
			setAnchorCalled = true
			if gotID != id {
				t.Fatalf("unexpected id: %d", gotID)
			}
			if txHash != "0xtx" || blockNumber != 7 {
				t.Fatalf("unexpected anchor: tx=%s block=%d", txHash, blockNumber)
			}
			return nil
		},
	}, &anchorMock{anchorFn: func(ctx context.Context, dataHashHex string) (string, uint64, error) {
		if len(dataHashHex) != 66 {
			t.Fatalf("unexpected hash length: %d", len(dataHashHex))
		}
		return "0xtx", 7, nil
	}})

	_, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !setAnchorCalled {
		t.Fatal("expected SetAnchorInfo to be called")
	}
}

func TestCreateAnchorFailureDoesNotBreakIngestion(t *testing.T) {
	const tsString = "2026-02-10T10:00:00Z"
	input := model.MeasurementIn{DeviceID: "sensor-1", Timestamp: tsString}

	svc := NewMeasurementService(&repoMock{
		createFn: func(ctx context.Context, in model.MeasurementIn, ts time.Time, dataHash string) (int64, error) {
			return 77, nil
		},
	}, &anchorMock{anchorFn: func(ctx context.Context, dataHashHex string) (string, uint64, error) {
		return "", 0, errors.New("rpc down")
	}})

	id, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 77 {
		t.Fatalf("expected id 77, got %d", id)
	}
}

func TestListValidation(t *testing.T) {
	svc := NewMeasurementService(&repoMock{}, nil)

	_, err := svc.List(context.Background(), "", "", "", "")
	if err == nil || err.Error() != "required query params: deviceId, from, to (RFC3339)" {
		t.Fatalf("expected required params error, got %v", err)
	}

	_, err = svc.List(context.Background(), "sensor-1", "2026-02-10T10:00:00Z", "2026-02-10T11:00:00Z", "0")
	if err == nil || err.Error() != "limit must be in range [1..5000]" {
		t.Fatalf("expected invalid limit error, got %v", err)
	}
}

func TestListSuccessMappingAndDefaultLimit(t *testing.T) {
	from := "2026-02-10T10:00:00Z"
	to := "2026-02-10T11:00:00Z"
	var seenLimit int

	temp := 10.5
	recordTS := time.Date(2026, 2, 10, 10, 30, 0, 0, time.FixedZone("EET", 2*3600))
	txHash := "0xtx"
	block := uint64(123)

	svc := NewMeasurementService(&repoMock{
		listFn: func(ctx context.Context, deviceID string, from, to time.Time, limit int) ([]model.MeasurementRecord, error) {
			seenLimit = limit
			if deviceID != "sensor-1" {
				t.Fatalf("unexpected device id: %s", deviceID)
			}
			return []model.MeasurementRecord{{
				ID:                7,
				DeviceID:          deviceID,
				Timestamp:         recordTS,
				Temperature:       &temp,
				DataHash:          "0xabc",
				AnchorTxHash:      &txHash,
				AnchorBlockNumber: &block,
			}}, nil
		},
	}, nil)

	out, err := svc.List(context.Background(), "sensor-1", from, to, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if seenLimit != 500 {
		t.Fatalf("expected default limit 500, got %d", seenLimit)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out))
	}
	if out[0].Timestamp != "2026-02-10T08:30:00Z" {
		t.Fatalf("expected UTC converted timestamp, got %s", out[0].Timestamp)
	}
	if out[0].DataHash != "0xabc" {
		t.Fatalf("expected data hash, got %s", out[0].DataHash)
	}
	if out[0].AnchorTxHash == nil || *out[0].AnchorTxHash != "0xtx" {
		t.Fatalf("expected anchor tx hash, got %v", out[0].AnchorTxHash)
	}
}
