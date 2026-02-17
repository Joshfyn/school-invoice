package handlers

/* 
import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/school-invoice/backend/internal/middleware"
	"github.com/school-invoice/backend/internal/models"
)

// GetSchoolProfile returns the current school's profile
func (h *Handler) GetSchoolProfile(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var school models.School
	err := h.dbx.QueryRowContext(ctx, `
		SELECT id, name, subdomain, phone, email, address, logo_url, is_active, created_at, updated_at
		FROM schools WHERE id = $1
	`, schoolID).Scan(
		&school.ID, &school.Name, &school.Subdomain, &school.Phone, &school.Email,
		&school.Address, &school.LogoURL, &school.IsActive, &school.CreatedAt, &school.UpdatedAt,
	)
	if err != nil {
		respondWithError(c, http.StatusNotFound, "not_found", "School not found")
		return
	}

	c.JSON(http.StatusOK, school.ToResponse())
}

// UpdateSchoolProfile updates the current school's profile
func (h *Handler) UpdateSchoolProfile(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)

	var req models.UpdateSchoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Build dynamic update query
	query := "UPDATE schools SET updated_at = $1"
	args := []interface{}{time.Now().UTC()}
	argIndex := 2

	if req.Name != nil {
		query += ", name = $" + string(rune('0'+argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Phone != nil {
		query += ", phone = $" + string(rune('0'+argIndex))
		args = append(args, *req.Phone)
		argIndex++
	}
	if req.Email != nil {
		query += ", email = $" + string(rune('0'+argIndex))
		args = append(args, *req.Email)
		argIndex++
	}
	if req.Address != nil {
		query += ", address = $" + string(rune('0'+argIndex))
		args = append(args, *req.Address)
		argIndex++
	}
	if req.LogoURL != nil {
		query += ", logo_url = $" + string(rune('0'+argIndex))
		args = append(args, *req.LogoURL)
		argIndex++
	}

	query += " WHERE id = $" + string(rune('0'+argIndex))
	args = append(args, schoolID)

	_, err := h.dbx.ExecContext(ctx, query, args...)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "database_error", "Failed to update school")
		return
	}

	// Return updated school
	h.GetSchoolProfile(c)
}
 */