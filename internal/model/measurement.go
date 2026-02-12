package model

import "time"

type MeasurementIn struct {
	DeviceID     string   `json:"deviceId"`
	Timestamp    string   `json:"timestamp"`
	Temperature  *float64 `json:"temperature,omitempty"`
	PH           *float64 `json:"ph,omitempty"`
	Turbidity    *float64 `json:"turbidity,omitempty"`
	Conductivity *float64 `json:"conductivity,omitempty"`
}

type MeasurementOut struct {
	ID           int64    `json:"id"`
	DeviceID     string   `json:"deviceId"`
	Timestamp    string   `json:"timestamp"`
	Temperature  *float64 `json:"temperature,omitempty"`
	PH           *float64 `json:"ph,omitempty"`
	Turbidity    *float64 `json:"turbidity,omitempty"`
	Conductivity *float64 `json:"conductivity,omitempty"`
}

type MeasurementRecord struct {
	ID           int64
	DeviceID     string
	Timestamp    time.Time
	Temperature  *float64
	PH           *float64
	Turbidity    *float64
	Conductivity *float64
}
