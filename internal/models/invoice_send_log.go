package models

import (
	"time"

	"github.com/google/uuid"
)

type InvoiceSendType string

const (
	InvoiceSendTypeInitial  InvoiceSendType = "initial"
	InvoiceSendTypeReminder InvoiceSendType = "reminder"
)

type InvoiceSendLog struct {
	BaseModel
	SchoolID  uuid.UUID       `json:"school_id" db:"school_id"`
	InvoiceID uuid.UUID       `json:"invoice_id" db:"invoice_id"`
	SentTo    string          `json:"sent_to" db:"sent_to"`
	SendType  InvoiceSendType `json:"send_type" db:"send_type"`
	SentBy    *uuid.UUID      `json:"sent_by,omitempty" db:"sent_by"`
}

type InvoiceSendLogResponse struct {
	ID        uuid.UUID       `json:"id"`
	InvoiceID uuid.UUID       `json:"invoice_id"`
	SentTo    string          `json:"sent_to"`
	SendType  InvoiceSendType `json:"send_type"`
	SentBy    *uuid.UUID      `json:"sent_by,omitempty"`
	CreatedAt string          `json:"created_at"`
}

func (l *InvoiceSendLog) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}

	rows, err := NamedQueryContext(dbx, ctx, `
		INSERT INTO invoice_send_logs (id, school_id, invoice_id, sent_to, send_type, sent_by)
		VALUES (:id, :school_id, :invoice_id, :sent_to, :send_type, :sent_by)
		RETURNING id, school_id, invoice_id, sent_to, send_type, sent_by, created_at
	`, l)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil
	}
	return rows.StructScan(l)
}

func ListInvoiceSendLogs(dbx DBTX, invoiceID uuid.UUID) ([]InvoiceSendLogResponse, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	logs := []InvoiceSendLog{}
	if err := dbx.SelectContext(ctx, &logs, `
		SELECT id, school_id, invoice_id, sent_to, send_type, sent_by, created_at
		FROM invoice_send_logs
		WHERE invoice_id = $1
		ORDER BY created_at DESC
	`, invoiceID); err != nil {
		return nil, err
	}

	resp := make([]InvoiceSendLogResponse, 0, len(logs))
	for _, l := range logs {
		resp = append(resp, InvoiceSendLogResponse{
			ID:        l.ID,
			InvoiceID: l.InvoiceID,
			SentTo:    l.SentTo,
			SendType:  l.SendType,
			SentBy:    l.SentBy,
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
		})
	}
	return resp, nil
}
