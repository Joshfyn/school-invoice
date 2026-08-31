package handlers

import (
	"net/url"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCreateGuardianPortalLinkUsesConfiguredDomain(t *testing.T) {
	const secret = "test-portal-secret"
	h := &Handler{config: &config.Config{
		JWTSecret:         secret,
		GuardianPortalURL: "https://guardians.staging.example.com/",
	}}
	schoolID := uuid.New()
	guardianID := uuid.New()
	invoiceID := uuid.New()

	link, err := h.createGuardianPortalLink(schoolID, guardianID, invoiceID)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(link, "https://guardians.staging.example.com/p/"))
	require.True(t, strings.HasSuffix(link, "/invoices/"+invoiceID.String()))

	parsedURL, err := url.Parse(link)
	require.NoError(t, err)
	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	require.Len(t, parts, 4)

	claims := &guardianPortalClaims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(_ *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	require.Equal(t, schoolID, claims.SchoolID)
	require.Equal(t, guardianID, claims.GuardianID)
	require.Equal(t, invoiceID, claims.InvoiceID)
	require.Equal(t, "guardian_portal", claims.Purpose)
}
