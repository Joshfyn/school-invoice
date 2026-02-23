package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/middleware"
	"github.com/school-invoice/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// Stub implementations for handlers
func (h *Handler) ListRoles(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	roles, err := models.ListRoles(h.dbx, schoolID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get roles")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get roles"})
		return
	}

	c.JSON(http.StatusOK, roles)
}

func (h *Handler) CreateRole(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	req := c.MustGet(middleware.ReqBodyCreateRole).(dto.CreateRoleRequest)

	role := models.Role{
		BaseModel: models.NewBaseModel(),
		SchoolID:  schoolID,
		Name:      req.Name,
		Description: func() string {
			if req.Description != "" {
				return req.Description
			}
			return ""
		}(),
		Permissions:  models.Permissions(req.Permissions),
		IsSuperAdmin: false,
	}
	if err := role.Create(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to create role")
		respondWithError(c, http.StatusInternalServerError, "server_error", "failed to create role")
		return
	}

	c.JSON(http.StatusCreated, role)
}

func (h *Handler) GetRole(c *gin.Context) {
	// TODO: If the role is super admin, return the role only if the schoolID is the same as the role's schoolID and own roleID
	// TODO: If the role is not super admin, return only the requestor's role
	schoolID := middleware.GetSchoolID(c)
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid_role_id", "invalid role id")
		return
	}

	role, err := models.GetRole(h.dbx, schoolID, roleID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get role")
		respondWithError(c, http.StatusNotFound, "not_found", "role not found")
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	req := c.MustGet(middleware.ReqBodyUpdateRole).(dto.UpdateRoleRequest)

	role, err := models.GetRole(h.dbx, schoolID, req.RoleID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get role for update")
		respondWithError(c, http.StatusNotFound, "not_found", "role not found")
		return
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Permissions != nil {
		role.Permissions = *req.Permissions
	}

	if err := role.Update(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to update role")
		respondWithError(c, http.StatusInternalServerError, "server_error", "failed to update role")
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *Handler) DeleteRole(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	req := c.MustGet(middleware.ReqBodyDeleteRole).(dto.DeleteRoleRequest)

	if err := models.DeleteRole(h.dbx, schoolID, req.RoleID); err != nil {
		h.logger.WithError(err).Error("Failed to delete role")
		respondWithError(c, http.StatusInternalServerError, "server_error", "failed to delete role")
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "role deleted"})
}

// User handlers
func (h *Handler) ListUsers(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	users, err := models.GetUsers(h.dbx, schoolID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get users")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *Handler) CreateUser(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateUser).(dto.CreateUserRequest)
	schoolID := middleware.GetSchoolID(c)

	// verify if the email already exists
	exists, err := (&models.User{Email: req.Email}).EmailExists(h.dbx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check email existence")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to check email existence"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Email already exists"})
		return
	}

	// verify school id is the same as the school id in the role
	role, err := models.GetRole(h.dbx, schoolID, req.RoleID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get role")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get role"})
		return
	}
	if role.SchoolID != schoolID {
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			WithField("role_id", req.RoleID).
			Error("School ID does not match role school ID")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "School ID does not match role school ID"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate password hash")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate password hash"})
		return
	}

	user := models.User{
		BaseModel:    models.NewBaseModel(),
		SchoolID:     schoolID,
		RoleID:       req.RoleID,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		IsActive:     func(v bool) *bool { return &v }(true),
	}

	if err := user.Create(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to create user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) GetSingleUser(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyGetSingleUser).(dto.GetSingleUserRequest)

	user := models.User{
		BaseModel: models.BaseModel{
			ID: req.UserID,
		},
		SchoolID: req.SchoolID,
	}
	userData, err := user.GetUser(h.dbx)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("user_id", user.BaseModel.ID).
			WithField("school_id", req.SchoolID).
			Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get user"})
		return
	}
	c.JSON(http.StatusOK, userData)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyUpdateUser).(dto.UpdateUserRequest)
	requestorSchoolID := middleware.GetSchoolID(c)

	user := models.User{
		BaseModel: models.BaseModel{
			ID: req.UserID,
		},
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
	}

	if err := user.Update(h.dbx, requestorSchoolID); err != nil {
		h.logger.WithError(err).Error("Failed to update user")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "User updated successfully"})
}

func (h *Handler) UpdateUserRole(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyUpdateUserRole).(dto.UpdateUserRoleRequest)
	requestorSchoolID := middleware.GetSchoolID(c)
	role, err := models.GetRole(h.dbx, requestorSchoolID, req.RoleID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get role")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get role"})
		return
	}
	if role.SchoolID != requestorSchoolID {
		h.logger.WithError(errors.New("unauthorized")).Error("Unauthorized: Role school ID does not match requestor school ID")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
		return
	}
	user := models.User{
		BaseModel: models.BaseModel{
			ID: req.UserID,
		},
		RoleID: req.RoleID,
	}
	if err := user.Update(h.dbx, requestorSchoolID); err != nil {
		h.logger.WithError(err).Error("Failed to update user role")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update user role"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "User role updated successfully"})
}

func (h *Handler) UpdateUserStatus(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyUpdateUserStatus).(dto.UpdateUserStatusRequest)
	requestorSchoolID := middleware.GetSchoolID(c)
	user := models.User{
		BaseModel: models.BaseModel{
			ID: req.UserID,
		},
		IsActive: req.IsActive,
	}
	if err := user.Update(h.dbx, requestorSchoolID); err != nil {
		h.logger.WithError(err).Error("Failed to update user status")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update user status"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "User status updated successfully"})
}

func (h *Handler) GetUserClassAccess(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get user class access - to be implemented"})
}

func (h *Handler) SetUserClassAccess(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Set user class access - to be implemented"})
}

// Session handlers
func (h *Handler) ListSessions(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	sessions, err := (&models.AcademicSession{SchoolID: schoolID}).List(h.dbx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list sessions")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list sessions"})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func (h *Handler) CreateSession(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateSession).(dto.CreateSessionRequest)

	session := models.AcademicSession{
		BaseModel: models.NewBaseModel(),
		SchoolID:  req.SchoolID,
		Name:      req.Name,
		StartDate: &req.Start,
		EndDate:   &req.End,
		IsCurrent: &req.IsCurrent,
	}
	if err := session.Create(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to create session")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (h *Handler) UpdateSession(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyUpdateSession).(dto.UpdateSessionRequest)
	session := models.AcademicSession{
		BaseModel: models.BaseModel{
			ID: req.SessionID,
		},
		SchoolID:  req.SchoolID,
		Name:      req.Name,
		StartDate: &req.Start,
		EndDate:   &req.End,
	}
	if err := session.Update(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to update session")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update session"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (h *Handler) SetCurrentSession(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodySetCurrentSession).(dto.SetCurrentSessionRequest)
	session := models.AcademicSession{
		BaseModel: models.BaseModel{
			ID: req.SessionID,
		},
		SchoolID:  req.SchoolID,
		IsCurrent: &req.IsCurrent,
	}
	if err := session.Update(h.dbx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_unique_current_session" {
			h.logger.WithError(err).Error("Another session is already current for this school")
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Another session is already current for this school"})
			return
		}
		h.logger.WithError(err).Error("Failed to set current session")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to set current session"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// Term handlers
func (h *Handler) ListTerms(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "List terms - to be implemented"})
}

func (h *Handler) CreateTerm(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateTerm).(dto.CreateTermRequest)
	term := models.Term{
		BaseModel: models.NewBaseModel(),
		SchoolID:  req.SchoolID,
		SessionID: req.SessionID,
		Name:      req.Name,
		SortOrder: req.SortOrder,
		IsCurrent: false,
	}
	if err := term.Create(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to create term")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create term"})
		return
	}

	c.JSON(http.StatusCreated, term)
}

func (h *Handler) UpdateTerm(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Update term - to be implemented"})
}

func (h *Handler) SetCurrentTerm(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Set current term - to be implemented"})
}

// Class handlers
func (h *Handler) ListClasses(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "List classes - to be implemented"})
}

func (h *Handler) CreateClass(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Create class - to be implemented"})
}

func (h *Handler) UpdateClass(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Update class - to be implemented"})
}

func (h *Handler) GetClassStudents(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get class students - to be implemented"})
}

// Enrollment handlers
func (h *Handler) ListEnrollments(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "List enrollments - to be implemented"})
}

func (h *Handler) CreateEnrollment(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Create enrollment - to be implemented"})
}

func (h *Handler) BulkEnrollment(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Bulk enrollment - to be implemented"})
}

func (h *Handler) UpdateEnrollmentStatus(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Update enrollment status - to be implemented"})
}

// Guardian handlers
func (h *Handler) ListGuardians(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "List guardians - to be implemented"})
}

func (h *Handler) CreateGuardian(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Create guardian - to be implemented"})
}

func (h *Handler) GetGuardian(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get guardian - to be implemented"})
}

func (h *Handler) UpdateGuardian(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Update guardian - to be implemented"})
}

func (h *Handler) GetGuardianInvoices(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get guardian invoices - to be implemented"})
}

// Student handlers
func (h *Handler) ListStudents(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "List students - to be implemented"})
}

func (h *Handler) CreateStudent(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Create student - to be implemented"})
}

func (h *Handler) BulkCreateStudents(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Bulk create students - to be implemented"})
}

func (h *Handler) GetStudent(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get student - to be implemented"})
}

func (h *Handler) UpdateStudent(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Update student - to be implemented"})
}

func (h *Handler) GetStudentEnrollments(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get student enrollments - to be implemented"})
}

func (h *Handler) GetStudentGuardians(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get student guardians - to be implemented"})
}

func (h *Handler) LinkStudentGuardian(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Link student guardian - to be implemented"})
}

// Fee type handlers
func (h *Handler) ListFeeTypes(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "List fee types - to be implemented"})
}

func (h *Handler) CreateFeeType(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Create fee type - to be implemented"})
}

func (h *Handler) UpdateFeeType(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Update fee type - to be implemented"})
}

func (h *Handler) GetFeeTypeAmounts(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get fee type amounts - to be implemented"})
}

func (h *Handler) SetFeeTypeAmounts(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Set fee type amounts - to be implemented"})
}

// Invoice handlers
func (h *Handler) ListInvoices(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "List invoices - to be implemented"})
}

func (h *Handler) CreateInvoice(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Create invoice - to be implemented"})
}

func (h *Handler) BulkCreateInvoices(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Bulk create invoices - to be implemented"})
}

func (h *Handler) GetInvoice(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get invoice - to be implemented"})
}

func (h *Handler) GetPublicInvoice(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get public invoice - to be implemented"})
}

func (h *Handler) SendInvoice(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Send invoice - to be implemented"})
}

func (h *Handler) GrantGrace(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Grant grace - to be implemented"})
}

func (h *Handler) SendReminder(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Send reminder - to be implemented"})
}

// Payment handlers
func (h *Handler) InitializePayment(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Initialize payment - to be implemented"})
}

func (h *Handler) PaymentWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Payment webhook - to be implemented"})
}

func (h *Handler) VerifyBankPayment(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Verify bank payment - to be implemented"})
}

func (h *Handler) GetReceipt(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get receipt - to be implemented"})
}

// Report handlers
func (h *Handler) GetDashboardSummary(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get dashboard summary - to be implemented"})
}

func (h *Handler) GetOutstandingReport(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get outstanding report - to be implemented"})
}

func (h *Handler) GetCollectionReport(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get collection report - to be implemented"})
}
