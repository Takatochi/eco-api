package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"eco-api/internal/model"
	"eco-api/internal/repository"
)

type MeasurementAnchor interface {
	Anchor(ctx context.Context, dataHashHex string) (txHash string, blockNumber uint64, err error)
}

type MeasurementService interface {
	Create(ctx context.Context, in model.MeasurementIn) (int64, error)
	List(ctx context.Context, deviceID, fromStr, toStr, limitStr string) ([]model.MeasurementOut, error)
}

type measurementService struct {
	repo   repository.MeasurementRepository
	anchor MeasurementAnchor
}

func NewMeasurementService(repo repository.MeasurementRepository, anchor MeasurementAnchor) MeasurementService {
	return &measurementService{repo: repo, anchor: anchor}
}

func (s *measurementService) Create(ctx context.Context, in model.MeasurementIn) (int64, error) {
	ts, err := validateAndParse(in)
	if err != nil {
		return 0, err
	}

	dataHash := measurementDataHash(in, ts)
	id, err := s.repo.Create(ctx, in, ts, dataHash)
	if err != nil {
		return 0, err
	}

	if s.anchor == nil {
		return id, nil
	}

	txHash, blockNumber, err := s.anchor.Anchor(ctx, dataHash)
	if err != nil {
		log.Printf("blockchain anchor failed for id=%d hash=%s: %v", id, dataHash, err)
		return id, nil
	}

	if err := s.repo.SetAnchorInfo(ctx, id, txHash, blockNumber); err != nil {
		log.Printf("persist blockchain anchor failed for id=%d tx=%s: %v", id, txHash, err)
	}

	return id, nil
}

func (s *measurementService) List(ctx context.Context, deviceID, fromStr, toStr, limitStr string) ([]model.MeasurementOut, error) {
	if deviceID == "" || fromStr == "" || toStr == "" {
		return nil, errors.New("required query params: deviceId, from, to (RFC3339)")
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return nil, errors.New("invalid from")
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return nil, errors.New("invalid to")
	}

	if !to.After(from) {
		return nil, errors.New("`to` must be after `from`")
	}

	limit := 500
	if limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v <= 0 || v > 5000 {
			return nil, errors.New("limit must be in range [1..5000]")
		}
		limit = v
	}

	records, err := s.repo.List(ctx, deviceID, from, to, limit)
	if err != nil {
		return nil, err
	}

	out := make([]model.MeasurementOut, 0, len(records))
	for _, r := range records {
		out = append(out, model.MeasurementOut{
			ID:                r.ID,
			DeviceID:          r.DeviceID,
			Timestamp:         r.Timestamp.UTC().Format(time.RFC3339),
			Temperature:       r.Temperature,
			PH:                r.PH,
			Turbidity:         r.Turbidity,
			Conductivity:      r.Conductivity,
			DataHash:          r.DataHash,
			AnchorTxHash:      r.AnchorTxHash,
			AnchorBlockNumber: r.AnchorBlockNumber,
		})
	}

	return out, nil
}

func validateAndParse(in model.MeasurementIn) (time.Time, error) {
	if in.DeviceID == "" {
		return time.Time{}, errors.New("deviceId is required")
	}

	if in.Timestamp == "" {
		return time.Time{}, errors.New("timestamp is required (RFC3339)")
	}

	ts, err := time.Parse(time.RFC3339, in.Timestamp)
	if err != nil {
		return time.Time{}, errors.New("timestamp must be RFC3339, example: 2026-02-10T10:00:00Z")
	}

	if in.PH != nil && (*in.PH < 0 || *in.PH > 14) {
		return time.Time{}, errors.New("ph must be in range [0..14]")
	}

	if in.Temperature != nil && (*in.Temperature < -50 || *in.Temperature > 80) {
		return time.Time{}, errors.New("temperature must be in range [-50..80]")
	}

	return ts, nil
}

func measurementDataHash(in model.MeasurementIn, ts time.Time) string {
	payload := fmt.Sprintf(
		"deviceId=%s|timestamp=%s|temperature=%s|ph=%s|turbidity=%s|conductivity=%s",
		in.DeviceID,
		ts.UTC().Format(time.RFC3339Nano),
		numberOrNil(in.Temperature),
		numberOrNil(in.PH),
		numberOrNil(in.Turbidity),
		numberOrNil(in.Conductivity),
	)
	digest := sha256.Sum256([]byte(payload))
	return "0x" + hex.EncodeToString(digest[:])
}

func numberOrNil(v *float64) string {
	if v == nil {
		return "nil"
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}
