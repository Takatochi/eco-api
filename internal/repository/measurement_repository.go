package repository

import (
	"context"
	"fmt"
	"time"

	"eco-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MeasurementRepository interface {
	Create(ctx context.Context, in model.MeasurementIn, ts time.Time, dataHash string) (int64, error)
	SetAnchorInfo(ctx context.Context, id int64, txHash string, blockNumber uint64) error
	List(ctx context.Context, deviceID string, from, to time.Time, limit int) ([]model.MeasurementRecord, error)
}

type measurementRepository struct {
	pool *pgxpool.Pool
}

func NewMeasurementRepository(pool *pgxpool.Pool) MeasurementRepository {
	return &measurementRepository{pool: pool}
}

func (r *measurementRepository) Create(ctx context.Context, in model.MeasurementIn, ts time.Time, dataHash string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO measurements (device_id, ts, temperature, ph, turbidity, conductivity, data_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, in.DeviceID, ts, in.Temperature, in.PH, in.Turbidity, in.Conductivity, dataHash).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create measurement: %w", err)
	}
	return id, nil
}

func (r *measurementRepository) SetAnchorInfo(ctx context.Context, id int64, txHash string, blockNumber uint64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE measurements
		SET anchor_tx_hash = $2, anchor_block_number = $3
		WHERE id = $1
	`, id, txHash, int64(blockNumber))
	if err != nil {
		return fmt.Errorf("set anchor info id=%d: %w", id, err)
	}
	return nil
}

func (r *measurementRepository) List(ctx context.Context, deviceID string, from, to time.Time, limit int) ([]model.MeasurementRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, device_id, ts, temperature, ph, turbidity, conductivity, data_hash, anchor_tx_hash, anchor_block_number
		FROM measurements
		WHERE device_id = $1 AND ts >= $2 AND ts <= $3
		ORDER BY ts ASC
		LIMIT $4
	`, deviceID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("list measurements: %w", err)
	}

	records, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.MeasurementRecord, error) {
		var m model.MeasurementRecord
		var blockNumber *int64
		if err := row.Scan(
			&m.ID,
			&m.DeviceID,
			&m.Timestamp,
			&m.Temperature,
			&m.PH,
			&m.Turbidity,
			&m.Conductivity,
			&m.DataHash,
			&m.AnchorTxHash,
			&blockNumber,
		); err != nil {
			return m, err
		}
		if blockNumber != nil {
			v := uint64(*blockNumber)
			m.AnchorBlockNumber = &v
		}
		return m, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan measurements: %w", err)
	}
	return records, nil
}
