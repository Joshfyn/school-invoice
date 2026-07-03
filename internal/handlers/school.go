package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/middleware"
	"github.com/school-invoice/backend/internal/models"
)

// GetSchoolProfile returns the current school's profile
func (h *Handler) GetSchoolProfile(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)

	school := models.School{BaseModel: models.BaseModel{ID: schoolID}}
	if err := school.GetProfile(h.dbx); err != nil {
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			Error("Failed to get school profile")
		c.JSON(http.StatusNotFound, "School not found")
		return
	}
	c.JSON(http.StatusOK, dto.SchoolResponse{
		ID:                school.ID,
		Name:              school.Name,
		Subdomain:         school.Subdomain,
		Phone:             school.Phone,
		Email:             school.Email,
		Address:           school.Address,
		LogoURL:           school.LogoURL,
		BankName:          school.BankName,
		BankAccountName:   school.BankAccountName,
		BankAccountNumber: school.BankAccountNumber,
		IsActive:          school.IsActive,
	})
}

// UpdateSchoolProfile updates the current school's profile
func (h *Handler) UpdateSchoolProfile(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	req := c.MustGet(middleware.ReqBodySchoolProfileUpdate).(dto.UpdateSchoolRequest)

	h.logger.
		WithField("school_id", schoolID).
		WithField("request", req).
		Info("Updating school profile")

	school := models.School{
		BaseModel: models.BaseModel{ID: schoolID},
	}
	if req.Name != nil {
		school.Name = *req.Name
	}
	if req.Phone != nil {
		school.Phone = *req.Phone
	}
	if req.Email != nil {
		school.Email = *req.Email
	}
	if req.Address != nil {
		school.Address = *req.Address
	}
	if req.LogoURL != nil {
		school.LogoURL = req.LogoURL
	}
	if req.BankName != nil {
		school.BankName = req.BankName
	}
	if req.BankAccountName != nil {
		school.BankAccountName = req.BankAccountName
	}
	if req.BankAccountNumber != nil {
		school.BankAccountNumber = req.BankAccountNumber
	}
	if err := school.Update(h.dbx); err != nil {
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			Error("Failed to update school profile")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "error",
			Message: "Failed to update school",
		})
		return
	}

	// Return updated school
	h.GetSchoolProfile(c)
}
