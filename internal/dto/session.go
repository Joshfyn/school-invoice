package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateSessionRequest struct {
	Name      string    `json:"-"`
	StartDate string    `json:"start_date" binding:"required"`
	EndDate   string    `json:"end_date" binding:"required"`
	IsCurrent bool      `json:"is_current"`
	Start     time.Time `json:"-"`
	End       time.Time `json:"-"`
	SchoolID  uuid.UUID `json:"-"`
}

type UpdateSessionRequest struct {
	SessionID uuid.UUID `json:"-"`
	Name      string    `json:"-"`
	SchoolID  uuid.UUID `json:"-"`
	StartDate string    `json:"start_date,omitempty"`
	EndDate   string    `json:"end_date,omitempty"`
	Start     time.Time `json:"-"`
	End       time.Time `json:"-"`
}

type SetCurrentSessionRequest struct {
	SessionID uuid.UUID `json:"-"`
	SchoolID  uuid.UUID `json:"-"`
	IsCurrent bool      `json:"is_current"`
}
