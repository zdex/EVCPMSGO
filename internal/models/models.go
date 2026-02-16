package models

import "time"

type Charger struct {
	ChargePointId string
	SecretHash    string
	IsActive      bool
	Vendor        string
	Model         string
	OcppVersion   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastSeenAt    *time.Time
}

type ConnectorState struct {
	ChargePointId string
	ConnectorId   int
	Status        string
	ErrorCode     string
	UpdatedAt     time.Time
}

type Session struct {
	SessionId     string
	ChargePointId string
	ConnectorId   int
	TransactionId int
	IdTag         string
	StartedAt     time.Time
	EndedAt       *time.Time
	MeterStartWh  *int64
	MeterStopWh   *int64
	Reason        *string
}

type MeterSample struct {
	Id            int64
	SessionId     string
	ChargePointId string
	TransactionId int
	Ts            time.Time
	SamplesJSON   []byte
}
