package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	guardianPortalTokenLifetime = 7 * 24 * time.Hour
	guardianPortalPurpose       = "guardian_portal"
)

type guardianPortalClaims struct {
	SchoolID   uuid.UUID `json:"school_id"`
	GuardianID uuid.UUID `json:"guardian_id"`
	InvoiceID  uuid.UUID `json:"invoice_id"`
	Purpose    string    `json:"purpose"`
	jwt.RegisteredClaims
}

// parseGuardianPortalToken validates a portal link token and returns its claims.
func (h *Handler) parseGuardianPortalToken(tokenString string) (*guardianPortalClaims, error) {
	claims := &guardianPortalClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(h.config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid portal token")
	}
	if claims.Purpose != guardianPortalPurpose {
		return nil, fmt.Errorf("token is not a guardian portal token")
	}
	if claims.SchoolID == uuid.Nil || claims.GuardianID == uuid.Nil {
		return nil, fmt.Errorf("portal token is missing school or guardian")
	}
	return claims, nil
}

func (h *Handler) createGuardianPortalLink(schoolID, guardianID, invoiceID uuid.UUID) (string, error) {
	if guardianID == uuid.Nil {
		return "", fmt.Errorf("guardian is required to create a portal link")
	}

	now := time.Now()
	claims := guardianPortalClaims{
		SchoolID:   schoolID,
		GuardianID: guardianID,
		InvoiceID:  invoiceID,
		Purpose:    guardianPortalPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   guardianID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(guardianPortalTokenLifetime)),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(h.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign guardian portal token: %w", err)
	}

	baseURL := strings.TrimRight(h.config.GuardianPortalURL, "/")
	return fmt.Sprintf(
		"%s/p/%s/invoices/%s",
		baseURL,
		url.PathEscape(token),
		invoiceID.String(),
	), nil
}
