package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/middleware"
	"github.com/school-invoice/backend/internal/models"
	"github.com/school-invoice/backend/lib/mail"
	pdflib "github.com/school-invoice/backend/lib/pdf"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

const (
	ConstraintUniqueCurrentTerm = "idx_unique_current_term"
	ErrorCodeUnique             = "23505"
)

func parseIntDefault(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return defaultValue
	}
	return parsed
}

// Stub implementations for handlers
func (h *Handler) ListRoles(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	roles, err := models.ListRoles(h.dbx, schoolID, middleware.GetIsSuperAdmin(c))
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
	schoolID := middleware.GetSchoolID(c)
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid_role_id", "invalid role id")
		return
	}

	role, err := models.GetRole(h.dbx, schoolID, roleID, middleware.GetIsSuperAdmin(c))
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

	role, err := models.GetRole(h.dbx, schoolID, req.RoleID, true)
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

	resp := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		roleResp := u.Role.ToResponse()
		isActive := u.User.IsActive != nil && *u.User.IsActive
		resp = append(resp, dto.UserResponse{
			ID:        u.User.ID,
			SchoolID:  u.User.SchoolID,
			RoleID:    u.User.RoleID,
			Email:     u.User.Email,
			FirstName: u.User.FirstName,
			LastName:  u.User.LastName,
			Phone:     u.User.Phone,
			IsActive:  isActive,
			Role:      &roleResp,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateUser(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateUser).(dto.CreateUserRequest)
	schoolID := middleware.GetSchoolID(c)
	isSuperAdmin := middleware.GetIsSuperAdmin(c)

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
	role, err := models.GetRole(h.dbx, schoolID, req.RoleID, true)
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
	if !isSuperAdmin && role.IsSuperAdmin {
		h.logger.WithError(err).Error("Only super admin can create super admin")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Only super admin can create super admin"})
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
	role, err := models.GetRole(h.dbx, requestorSchoolID, req.RoleID, true)
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
	schoolID := middleware.GetSchoolID(c)

	terms, err := (&models.Term{SchoolID: schoolID}).List(h.dbx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list terms")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list terms"})
		return
	}
	c.JSON(http.StatusOK, terms)
}

func (h *Handler) CreateTerm(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateTerm).(dto.CreateTermRequest)
	isCurrent := false
	term := models.Term{
		BaseModel: models.NewBaseModel(),
		SchoolID:  req.SchoolID,
		SessionID: req.SessionID,
		Name:      req.Name,
		SortOrder: req.SortOrder,
		StartDate: &req.Start,
		EndDate:   &req.End,
		IsCurrent: &isCurrent,
	}
	if err := term.Create(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to create term")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create term"})
		return
	}

	c.JSON(http.StatusCreated, term)
}

func (h *Handler) UpdateTerm(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyUpdateTerm).(dto.UpdateTermRequest)
	term := models.Term{
		BaseModel: models.BaseModel{
			ID: req.TermID,
		},
		SchoolID:  req.SchoolID,
		SessionID: req.SessionID,
		Name:      req.Name,
		SortOrder: req.SortOrder,
		StartDate: &req.Start,
		EndDate:   &req.End,
	}
	if err := term.Update(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to update term")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update term"})
		return
	}

	c.JSON(http.StatusOK, term)
}

func (h *Handler) SetCurrentTerm(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodySetCurrentTerm).(dto.SetCurrentTermRequest)

	if req.IsCurrent {
		if err := models.ClearCurrentTerm(h.dbx, req.SchoolID); err != nil {
			h.logger.
				WithError(err).
				WithField("school_id", req.SchoolID).
				Error("Failed to clear current term")
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to set current term"})
			return
		}
	}

	isCurrent := req.IsCurrent
	term := models.Term{
		BaseModel: models.BaseModel{
			ID: req.TermID,
		},
		SchoolID:  req.SchoolID,
		IsCurrent: &isCurrent,
	}
	if err := term.Update(h.dbx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == ErrorCodeUnique && pgErr.ConstraintName == ConstraintUniqueCurrentTerm {
			h.logger.WithError(err).
				WithField("school_id", req.SchoolID).
				Error("Another term is already current for this school")
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Another term is already current for this school"})
			return
		}
		h.logger.WithError(err).
			WithField("school_id", req.SchoolID).
			WithField("term_id", req.TermID).
			Error("Failed to set current term")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to set current term"})
		return
	}

	terms, err := (&models.Term{SchoolID: req.SchoolID}).List(h.dbx)
	if err != nil {
		c.JSON(http.StatusOK, term)
		return
	}
	for _, t := range terms {
		if t.ID == req.TermID {
			c.JSON(http.StatusOK, t)
			return
		}
	}

	c.JSON(http.StatusOK, term)
}

const constraintUniqueClass = "idx_unique_class"

// Class handlers
func (h *Handler) ListClasses(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	level := c.Query("level")

	classes, err := models.ListClasses(h.dbx, schoolID, level)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list classes")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list classes"})
		return
	}

	resp := make([]models.ClassResponse, 0, len(classes))
	for i := range classes {
		resp = append(resp, classes[i].ToResponse())
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateClass(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateClass).(dto.CreateClassRequest)
	class := models.Class{
		BaseModel: models.NewBaseModel(),
		SchoolID:  req.SchoolID,
		Name:      req.Name,
		Section:   req.Section,
		SortOrder: req.SortOrder,
		IsActive:  true,
	}

	if err := class.Create(h.dbx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == ErrorCodeUnique && pgErr.ConstraintName == constraintUniqueClass {
			h.logger.WithError(err).
				WithField("user_id", req.UserID).
				WithField("school_id", req.SchoolID).
				WithField("name", req.Name).
				WithField("section", req.Section).
				Error("Class already exists")
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Class already exists"})
			return
		}
		h.logger.WithError(err).
			WithField("user_id", req.UserID).
			WithField("school_id", req.SchoolID).
			WithField("name", req.Name).
			WithField("section", req.Section).
			Error("Failed to create class")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create class"})
		return
	}

	c.JSON(http.StatusCreated, class)
}

func (h *Handler) UpdateClass(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyUpdateClass).(dto.UpdateClassRequest)

	class := models.Class{
		BaseModel: models.BaseModel{ID: req.ClassID},
		SchoolID:  req.SchoolID,
		Section:   req.Section,
		SortOrder: req.SortOrder,
		IsActive:  req.IsActive,
	}
	if req.Name != "" {
		class.Name = string(req.Name)
	}

	if err := class.Update(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to update class")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update class"})
		return
	}

	if err := class.Get(h.dbx, req.SchoolID); err != nil {
		h.logger.WithError(err).Error("Failed to get updated class")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get updated class"})
		return
	}

	c.JSON(http.StatusOK, class.ToResponse())
}

func (h *Handler) GetClassStudents(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	classID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid_class_id", "invalid class id")
		return
	}

	students, err := models.ListClassStudents(h.dbx, schoolID, classID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get class students")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get class students"})
		return
	}

	c.JSON(http.StatusOK, students)
}

// Enrollment handlers
func (h *Handler) ListEnrollments(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	filters := models.EnrollmentFilters{}

	if termID := c.Query("term_id"); termID != "" {
		id, err := uuid.Parse(termID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid term_id"})
			return
		}
		filters.TermID = &id
	}
	if classID := c.Query("class_id"); classID != "" {
		id, err := uuid.Parse(classID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid class_id"})
			return
		}
		filters.ClassID = &id
	}
	if status := c.Query("status"); status != "" {
		s := models.EnrollmentStatus(status)
		filters.Status = &s
	}

	enrollments, err := models.ListEnrollments(h.dbx, schoolID, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list enrollments")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list enrollments"})
		return
	}

	resp := make([]models.EnrollmentResponse, 0, len(enrollments))
	for i := range enrollments {
		resp = append(resp, enrollments[i].ToResponse())
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateEnrollment(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	var req models.CreateEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	enrollment := models.StudentEnrollment{
		BaseModel:  models.NewBaseModel(),
		SchoolID:   schoolID,
		StudentID:  req.StudentID,
		ClassID:    req.ClassID,
		TermID:     req.TermID,
		Status:     models.EnrollmentActive,
		EnrolledAt: time.Now().UTC(),
	}
	if err := enrollment.Create(h.dbx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == ErrorCodeUnique {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Student already enrolled for this term"})
			return
		}
		h.logger.WithError(err).Error("Failed to create enrollment")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create enrollment"})
		return
	}

	c.JSON(http.StatusCreated, enrollment.ToResponse())
}

func (h *Handler) BulkEnrollment(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	var req models.BulkEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	count, err := models.BulkEnrollStudents(h.dbx, schoolID, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to bulk enroll students")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to bulk enroll students"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"enrolled_count": count})
}

func (h *Handler) UpdateEnrollmentStatus(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	enrollmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid enrollment id"})
		return
	}

	var req models.UpdateEnrollmentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	enrollment := models.StudentEnrollment{
		BaseModel: models.BaseModel{ID: enrollmentID},
		SchoolID:  schoolID,
		Status:    req.Status,
	}
	if err := enrollment.UpdateStatus(h.dbx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Enrollment not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to update enrollment status")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update enrollment status"})
		return
	}

	if err := enrollment.Get(h.dbx, schoolID); err != nil {
		h.logger.WithError(err).Error("Failed to get enrollment")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get enrollment"})
		return
	}

	c.JSON(http.StatusOK, enrollment.ToResponse())
}

// Guardian handlers
func (h *Handler) ListGuardians(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)

	guardians, err := models.ListGuardiansForSchool(h.dbx, schoolID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list guardians")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list guardians"})
		return
	}

	c.JSON(http.StatusOK, guardians)
}

func (h *Handler) CreateGuardian(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateGuardian).(dto.CreateGuardianRequest)
	schoolID := middleware.GetSchoolID(c)

	admitted, err := models.StudentAdmittedToSchool(h.dbx, schoolID, req.StudentID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to verify student")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to verify student"})
		return
	}
	if !admitted {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Student is not enrolled in this school"})
		return
	}

	guardian := models.Guardian{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Email:     req.Email,
		Address:   req.Address,
	}

	exists, err := (&models.Guardian{Phone: req.Phone}).PhoneExists(h.dbx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check guardian existence")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to check guardian existence"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Guardian with this phone number already exists"})
		return
	}

	if req.Email != "" {
		exists, err = (&models.Guardian{Email: req.Email}).EmailExists(h.dbx)
		if err != nil {
			h.logger.WithError(err).Error("Failed to check guardian existence")
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to check guardian existence"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Guardian with this email already exists"})
			return
		}
	}

	tx := h.dbx.MustBegin()
	defer tx.Rollback()

	if err := guardian.Create(tx); err != nil {
		h.logger.WithError(err).Error("Failed to create guardian")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create guardian"})
		return
	}

	link := models.StudentGuardian{
		BaseModel:             models.NewBaseModel(),
		StudentID:             req.StudentID,
		GuardianID:            guardian.ID,
		Relationship:          models.Relationship(req.Relationship),
		IsPrimary:             req.IsPrimary,
		ReceivesNotifications: req.ReceivesNotifications,
	}
	if err := link.Create(tx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == ErrorCodeUnique {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Guardian already linked to this student"})
			return
		}
		h.logger.WithError(err).Error("Failed to link guardian to student")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to link guardian to student"})
		return
	}

	if err := tx.Commit(); err != nil {
		h.logger.WithError(err).Error("Failed to commit guardian creation")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create guardian"})
		return
	}

	detail, err := models.GetGuardianWithStudents(h.dbx, schoolID, guardian.ID)
	if err != nil {
		c.JSON(http.StatusCreated, guardian)
		return
	}
	c.JSON(http.StatusCreated, detail)
}

func (h *Handler) GetGuardian(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	guardianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid guardian id"})
		return
	}

	detail, err := models.GetGuardianWithStudents(h.dbx, schoolID, guardianID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Guardian not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get guardian")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get guardian"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handler) UpdateGuardian(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyUpdateGuardian).(dto.UpdateGuardianRequest)

	if req.Phone != "" {
		check := models.Guardian{ID: req.GuardianID, Phone: req.Phone}
		if exists, err := check.PhoneExistsForOther(h.dbx); err != nil {
			h.logger.WithError(err).Error("Failed to check guardian existence")
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to check guardian existence"})
			return
		} else if exists {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Guardian phone number already exists"})
			return
		}
	}

	if req.Email != "" {
		check := models.Guardian{ID: req.GuardianID, Email: req.Email}
		if exists, err := check.EmailExistsForOther(h.dbx); err != nil {
			h.logger.WithError(err).Error("Failed to check guardian existence")
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to check guardian existence"})
			return
		} else if exists {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Guardian email already exists"})
			return
		}
	}

	guardian := models.Guardian{
		ID:        req.GuardianID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Email:     req.Email,
		Address:   req.Address,
	}
	if err := guardian.Update(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to update guardian")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update guardian"})
		return
	}
	c.JSON(http.StatusOK, guardian)
}

func (h *Handler) GetGuardianInvoices(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	guardianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid guardian id"})
		return
	}

	invoices, err := models.ListGuardianInvoices(h.dbx, schoolID, guardianID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get guardian invoices")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get guardian invoices"})
		return
	}

	resp := make([]models.InvoiceResponse, 0, len(invoices))
	for i := range invoices {
		resp = append(resp, invoices[i].ToResponse())
	}
	c.JSON(http.StatusOK, resp)
}

// Student handlers
func (h *Handler) ListStudents(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	filters := models.StudentListFilters{
		Search: c.Query("search"),
		Page:   parseIntDefault(c.Query("page"), 1),
		Limit:  parseIntDefault(c.Query("limit"), 20),
	}
	if classID := c.Query("class_id"); classID != "" {
		id, err := uuid.Parse(classID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid class_id"})
			return
		}
		filters.ClassID = &id
	}

	students, total, err := models.ListStudents(h.dbx, schoolID, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list students")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list students"})
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponse(students, filters.Page, filters.Limit, total))
}

func (h *Handler) CreateStudent(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateStudent).(dto.CreateStudentRequest)
	schoolID := middleware.GetSchoolID(c)
	student := models.Student{
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		Gender:      req.Gender,
		DateOfBirth: req.DateOfBirth,
		NIN:         req.NIN,
	}
	// check if the student NIN already exists
	exists, err := student.NINExists(h.dbx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check student existence")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to check student existence"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Student NIN already exists"})
		return
	}

	tx := h.dbx.MustBegin()
	defer tx.Rollback()

	if err := student.Create(tx); err != nil {
		h.logger.
			WithError(err).
			WithField("first_name", req.FirstName).
			WithField("middle_name", req.MiddleName).
			WithField("nin", req.NIN).
			Error("Failed to create student")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create student"})
		return
	}

	admission := models.StudentAdmission{
		StudentID:     student.ID,
		SchoolID:      schoolID,
		AdmissionNo:   models.GenerateAdmissionNo(schoolID.String(), student.ID.String()),
		AdmissionDate: time.Now().UTC(),
	}
	if err := admission.Create(tx); err != nil {
		h.logger.WithError(err).WithField("student_id", student.ID).Error("Failed to create student admission")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create student admission"})
		return
	}

	if err := tx.Commit(); err != nil {
		h.logger.WithError(err).Error("Failed to commit student creation")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create student"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            student.ID,
		"first_name":    student.FirstName,
		"middle_name":   student.MiddleName,
		"last_name":     student.LastName,
		"gender":        student.Gender,
		"date_of_birth": student.DateOfBirth,
		"nin":           student.NIN,
		"admission_no":  admission.AdmissionNo,
		"created_at":    student.CreatedAt,
		"updated_at":    student.UpdatedAt,
	})
}

func (h *Handler) BulkCreateStudents(c *gin.Context) {
	c.JSON(http.StatusCreated, models.SuccessResponse{Message: "Bulk create students - to be implemented"})
}

func (h *Handler) GetStudent(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyGetSingleStudent).(dto.GetSingleStudentRequest)
	student := models.Student{
		ID: req.StudentID,
	}
	err := student.Get(h.dbx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.
				WithError(err).
				WithField("student_id", req.StudentID).
				Error("Student not found")
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Student not found"})
			return
		}
		h.logger.
			WithError(err).
			WithField("student_id", req.StudentID).
			Error("Failed to get student")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get student"})
		return
	}
	c.JSON(http.StatusOK, student)
}

func (h *Handler) UpdateStudent(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyUpdateStudent).(dto.UpdateStudentRequest)
	student := models.Student{
		ID: req.StudentID,
	}
	if req.FirstName != "" {
		student.FirstName = req.FirstName
	}
	if req.MiddleName != "" {
		student.MiddleName = req.MiddleName
	}
	if req.LastName != "" {
		student.LastName = req.LastName
	}
	if req.Gender != "" {
		student.Gender = req.Gender
	}
	if !req.DateOfBirth.IsZero() {
		student.DateOfBirth = &req.DateOfBirth
	}
	if req.NIN != "" {
		student.NIN = req.NIN
	}

	if err := student.Update(h.dbx); err != nil {
		h.logger.
			WithError(err).
			WithField("student_id", req.StudentID).
			Error("Failed to update student")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update student"})
		return
	}
	c.JSON(http.StatusOK, student)
}

func (h *Handler) CreateStudentAdmission(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyCreateStudentAdmission).(dto.CreateStudentAdmissionRequest)
	admissionDate, err := time.Parse(middleware.DateFormat, req.AdmissionDate)
	if err != nil {
		h.logger.
			WithError(err).
			Error("Failed to parse admission date")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to parse admission date"})
		return
	}
	studentAdmission := models.StudentAdmission{
		StudentID:     req.StudentID,
		SchoolID:      req.SchoolID,
		AdmissionNo:   req.AdmissionNo,
		AdmissionDate: admissionDate,
	}
	if err := studentAdmission.Create(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to create student admission")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create student admission"})
		return
	}
	c.JSON(http.StatusCreated, studentAdmission)
}

func (h *Handler) DeleteStudentAdmission(c *gin.Context) {
	req := c.MustGet(middleware.ReqBodyDeleteStudentAdmission).(dto.DeleteStudentAdmissionRequest)
	studentAdmission := models.StudentAdmission{
		StudentID: req.StudentID,
		SchoolID:  req.SchoolID,
	}
	deleted, err := studentAdmission.Delete(h.dbx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete student admission")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to delete student admission"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Student admission not found"})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Student admission deleted"})
}

func (h *Handler) GetStudentEnrollments(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid student id"})
		return
	}

	enrollments, err := models.ListStudentEnrollments(h.dbx, schoolID, studentID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get student enrollments")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get student enrollments"})
		return
	}

	resp := make([]models.EnrollmentResponse, 0, len(enrollments))
	for i := range enrollments {
		resp = append(resp, enrollments[i].ToResponse())
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetStudentGuardians(c *gin.Context) {
	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid student id"})
		return
	}

	guardians, err := models.ListStudentGuardians(h.dbx, studentID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get student guardians")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get student guardians"})
		return
	}

	c.JSON(http.StatusOK, guardians)
}

func (h *Handler) LinkStudentGuardian(c *gin.Context) {
	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid student id"})
		return
	}

	var req models.LinkGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	link := models.StudentGuardian{
		BaseModel:             models.NewBaseModel(),
		StudentID:             studentID,
		GuardianID:            req.GuardianID,
		Relationship:          req.Relationship,
		IsPrimary:             req.IsPrimary,
		ReceivesNotifications: req.ReceivesNotifications,
	}
	if err := link.Create(h.dbx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == ErrorCodeUnique {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Guardian already linked to student"})
			return
		}
		h.logger.WithError(err).Error("Failed to link student guardian")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to link student guardian"})
		return
	}

	c.JSON(http.StatusCreated, link)
}

// Fee type handlers
func (h *Handler) ListFeeTypes(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	activeOnly := c.Query("active_only") == "true"

	feeTypes, err := models.ListFeeTypes(h.dbx, schoolID, activeOnly)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list fee types")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list fee types"})
		return
	}

	resp := make([]models.FeeTypeResponse, 0, len(feeTypes))
	for i := range feeTypes {
		resp = append(resp, feeTypes[i].ToResponse())
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateFeeType(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	var req models.CreateFeeTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	exists, err := models.FeeCategoryExists(h.dbx, schoolID, string(req.Category))
	if err != nil {
		h.logger.WithError(err).Error("Failed to validate fee category")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to validate fee category"})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid fee category"})
		return
	}

	feeType := models.FeeType{
		BaseModel:     models.NewBaseModel(),
		SchoolID:      schoolID,
		Name:          req.Name,
		Description:   req.Description,
		DefaultAmount: decimal.NewFromFloat(req.DefaultAmount),
		Category:      req.Category,
		Frequency:     req.Frequency,
		IsOptional:    req.IsOptional,
		IsActive:      true,
	}
	if err := feeType.Create(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to create fee type")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create fee type"})
		return
	}

	c.JSON(http.StatusCreated, feeType.ToResponse())
}

func (h *Handler) UpdateFeeType(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	feeTypeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid fee type id"})
		return
	}

	var req models.UpdateFeeTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	feeType := models.FeeType{BaseModel: models.BaseModel{ID: feeTypeID}, SchoolID: schoolID}
	if err := feeType.Get(h.dbx, schoolID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Fee type not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get fee type")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get fee type"})
		return
	}

	if req.Name != nil {
		feeType.Name = *req.Name
	}
	if req.Description != nil {
		feeType.Description = *req.Description
	}
	if req.DefaultAmount != nil {
		feeType.DefaultAmount = decimal.NewFromFloat(*req.DefaultAmount)
	}
	if req.Category != nil {
		exists, err := models.FeeCategoryExists(h.dbx, schoolID, string(*req.Category))
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid fee category"})
			return
		}
		feeType.Category = *req.Category
	}
	if req.Frequency != nil {
		feeType.Frequency = *req.Frequency
	}
	if req.IsOptional != nil {
		feeType.IsOptional = *req.IsOptional
	}
	if req.IsActive != nil {
		feeType.IsActive = *req.IsActive
	}

	if err := feeType.Update(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to update fee type")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update fee type"})
		return
	}

	c.JSON(http.StatusOK, feeType.ToResponse())
}

func (h *Handler) GetFeeTypeAmounts(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	feeTypeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid fee type id"})
		return
	}

	feeType := models.FeeType{BaseModel: models.BaseModel{ID: feeTypeID}}
	if err := feeType.Get(h.dbx, schoolID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Fee type not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get fee type")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get fee type"})
		return
	}

	amounts, err := models.ListFeeClassAmounts(h.dbx, feeTypeID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get fee type amounts")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get fee type amounts"})
		return
	}

	resp := make([]models.FeeClassAmountResponse, 0, len(amounts))
	for i := range amounts {
		item := amounts[i].ToResponse()
		class := models.Class{BaseModel: models.BaseModel{ID: amounts[i].ClassID}}
		if err := class.Get(h.dbx, schoolID); err == nil {
			classResp := class.ToResponse()
			item.Class = &classResp
		}
		resp = append(resp, item)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SetFeeTypeAmounts(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	feeTypeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid fee type id"})
		return
	}

	var req models.SetFeeClassAmountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	feeType := models.FeeType{BaseModel: models.BaseModel{ID: feeTypeID}}
	if err := feeType.Get(h.dbx, schoolID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Fee type not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get fee type")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get fee type"})
		return
	}

	if err := models.SetFeeClassAmounts(h.dbx, feeTypeID, req.Amounts); err != nil {
		h.logger.WithError(err).Error("Failed to set fee type amounts")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to set fee type amounts"})
		return
	}

	h.GetFeeTypeAmounts(c)
}

func (h *Handler) DeleteFeeType(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	feeTypeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid fee type id"})
		return
	}

	feeType := models.FeeType{BaseModel: models.BaseModel{ID: feeTypeID}, SchoolID: schoolID}
	if err := feeType.Get(h.dbx, schoolID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Fee type not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get fee type")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get fee type"})
		return
	}

	if err := feeType.Delete(h.dbx); err != nil {
		if errors.Is(err, models.ErrFeeTypeInUse) {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Fee type is used on invoices and cannot be deleted"})
			return
		}
		h.logger.WithError(err).Error("Failed to delete fee type")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to delete fee type"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Fee type deleted"})
}

func (h *Handler) ListFeeCategories(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	categories, err := models.ListFeeCategories(h.dbx, schoolID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list fee categories")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list fee categories"})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *Handler) CreateFeeCategory(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	var req models.CreateFeeCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	code := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Code), " ", "_"))
	fc := models.FeeCategoryRecord{
		BaseModel: models.NewBaseModel(),
		SchoolID:  schoolID,
		Name:      strings.TrimSpace(req.Name),
		Code:      code,
		IsActive:  true,
	}
	if err := fc.Create(h.dbx); err != nil {
		h.logger.WithError(err).Error("Failed to create fee category")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create fee category"})
		return
	}
	c.JSON(http.StatusCreated, fc)
}

func (h *Handler) DeleteFeeCategory(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid category id"})
		return
	}

	categories, err := models.ListFeeCategories(h.dbx, schoolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list fee categories"})
		return
	}

	var target *models.FeeCategoryRecord
	for i := range categories {
		if categories[i].ID == categoryID {
			target = &categories[i]
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Fee category not found"})
		return
	}

	if err := target.Delete(h.dbx, schoolID); err != nil {
		if errors.Is(err, models.ErrFeeCategoryInUse) {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Category is used by fee types and cannot be deleted"})
			return
		}
		h.logger.WithError(err).Error("Failed to delete fee category")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to delete fee category"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Fee category deleted"})
}

// Invoice handlers
func (h *Handler) ListInvoices(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	filters := models.InvoiceListFilters{
		Search: c.Query("search"),
		Page:   parseIntDefault(c.Query("page"), 1),
		Limit:  parseIntDefault(c.Query("limit"), 20),
	}
	if status := c.Query("status"); status != "" {
		s := models.InvoiceStatus(status)
		filters.Status = &s
	}
	if classID := c.Query("class_id"); classID != "" {
		id, err := uuid.Parse(classID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid class_id"})
			return
		}
		filters.ClassID = &id
	}
	if termID := c.Query("term_id"); termID != "" {
		id, err := uuid.Parse(termID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid term_id"})
			return
		}
		filters.TermID = &id
	}

	invoices, total, err := models.ListInvoices(h.dbx, schoolID, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list invoices")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list invoices"})
		return
	}

	resp := make([]models.InvoiceResponse, 0, len(invoices))
	invoiceIDs := make([]uuid.UUID, 0, len(invoices))
	for i := range invoices {
		resp = append(resp, invoices[i].ToResponse())
		invoiceIDs = append(invoiceIDs, invoices[i].ID)
	}

	if len(invoiceIDs) > 0 {
		lastSent, err := models.GetLastSentAtByInvoices(h.dbx, invoiceIDs)
		if err == nil {
			for i := range resp {
				if t, ok := lastSent[resp[i].ID]; ok {
					s := t.Format(time.RFC3339)
					resp[i].LastSentAt = &s
				}
			}
		}
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponse(resp, filters.Page, filters.Limit, total))
}

func (h *Handler) CreateInvoice(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	var req models.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			Error("Failed to bind JSON for create invoice")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	invoice, items, err := models.BuildInvoiceFromRequest(h.dbx, schoolID, req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.
				WithError(err).
				WithField("school_id", schoolID).
				WithField("student_id", req.StudentID).
				WithField("fee_type_ids", req.FeeTypeIDs).
				Error("Student has no active enrollment for invoice")
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "no_active_enrollment",
				Message: "Student has no active enrollment in the current term",
			})
			return
		}
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			WithField("student_id", req.StudentID).
			WithField("fee_type_ids", req.FeeTypeIDs).
			Error("Failed to build invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to build invoice"})
		return
	}

	if err := models.CreateInvoiceWithItems(h.dbx, invoice, items); err != nil {
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			WithField("student_id", req.StudentID).
			WithField("invoice_no", invoice.InvoiceNo).
			Error("Failed to create invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create invoice"})
		return
	}

	resp, err := models.GetInvoiceWithItems(h.dbx, schoolID, invoice.ID)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			WithField("invoice_id", invoice.ID).
			Error("Failed to get created invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get created invoice"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) BulkCreateInvoices(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	var req models.BulkCreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.
			WithError(err).
			WithField("school_id", schoolID).
			Error("Failed to bind JSON for bulk create invoices")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	studentIDs := req.StudentIDs
	if req.ClassID != nil && len(studentIDs) == 0 {
		students, err := models.ListClassStudents(h.dbx, schoolID, *req.ClassID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to list class students for bulk invoice")
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list class students"})
			return
		}
		for _, s := range students {
			studentIDs = append(studentIDs, s.ID)
		}
	}

	created := []models.InvoiceResponse{}
	for _, studentID := range studentIDs {
		invoiceReq := models.CreateInvoiceRequest{
			StudentID:  studentID,
			FeeTypeIDs: req.FeeTypeIDs,
			DueDate:    req.DueDate,
		}
		invoice, items, err := models.BuildInvoiceFromRequest(h.dbx, schoolID, invoiceReq)
		if err != nil {
			continue
		}
		if err := models.CreateInvoiceWithItems(h.dbx, invoice, items); err != nil {
			continue
		}
		resp, err := models.GetInvoiceWithItems(h.dbx, schoolID, invoice.ID)
		if err == nil {
			created = append(created, *resp)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"created_count": len(created), "invoices": created})
}

func (h *Handler) GetInvoice(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid invoice id"})
		return
	}

	resp, err := models.GetInvoiceDetail(h.dbx, schoolID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get invoice"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) sendInvoiceEmail(c *gin.Context, invoiceID uuid.UUID, sendType models.InvoiceSendType, recipientEmail string) {
	schoolID := middleware.GetSchoolID(c)
	userID := middleware.GetUserID(c)

	ctx, err := models.GetInvoicePDFContext(h.dbx, schoolID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.WithError(err).
				WithField("school_id", schoolID).
				WithField("invoice_id", invoiceID).
				Error("Invoice not found")
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "invoice_not_found",
				Message: "Invoice not found",
			})
			return
		}
		h.logger.WithError(err).
			WithField("school_id", schoolID).
			WithField("invoice_id", invoiceID).
			Error("Failed to load invoice for sending")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load invoice",
		})
		return
	}

	sendTo := strings.TrimSpace(recipientEmail)
	if sendTo == "" {
		sendTo = strings.TrimSpace(ctx.GuardianEmail)
	}
	if sendTo == "" {
		h.logger.
			WithField("school_id", schoolID).
			WithField("invoice_id", invoiceID).
			Error("No recipient email for invoice send")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "no_recipient_email",
			Message: "Provide an email address or add an email to the student's guardian.",
		})
		return
	}

	pdfData := pdflib.FromInvoiceContext(ctx)
	pdfBytes, err := pdflib.GenerateInvoicePDF(pdfData)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate invoice PDF")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate invoice PDF"})
		return
	}

	isReminder := sendType == models.InvoiceSendTypeReminder
	if err := mail.SendInvoiceEmail(sendTo, ctx.School.Name, ctx.Invoice.InvoiceNo, ctx.StudentName, pdfBytes, isReminder); err != nil {
		h.logger.WithError(err).Error("Failed to send invoice email")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to send invoice email", Message: err.Error()})
		return
	}

	sendLog := models.InvoiceSendLog{
		BaseModel: models.NewBaseModel(),
		SchoolID:  schoolID,
		InvoiceID: invoiceID,
		SentTo:    sendTo,
		SendType:  sendType,
		SentBy:    &userID,
	}
	if err := sendLog.Create(h.dbx); err != nil {
		h.logger.WithError(err).Warn("Invoice sent but failed to record send log")
	}

	resp, err := models.GetInvoiceDetail(h.dbx, schoolID, invoiceID)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse{Message: "Invoice sent successfully"})
		return
	}

	msg := "Invoice sent successfully"
	if isReminder {
		msg = "Invoice reminder sent successfully"
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: msg, Data: resp})
}

func (h *Handler) GetPublicInvoice(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Get public invoice - to be implemented"})
}

func (h *Handler) SendInvoice(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid invoice id"})
		return
	}
	var req models.SendInvoiceRequest
	_ = c.ShouldBindJSON(&req)
	h.sendInvoiceEmail(c, invoiceID, models.InvoiceSendTypeInitial, req.Email)
}

func (h *Handler) UpdateInvoiceStatus(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid invoice id"})
		return
	}

	var req models.UpdateInvoiceStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	invoice := models.Invoice{BaseModel: models.BaseModel{ID: invoiceID}, SchoolID: schoolID}
	if err := invoice.Get(h.dbx, schoolID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get invoice"})
		return
	}

	invoice.Status = req.Status
	if err := invoice.UpdateStatus(h.dbx); err != nil {
		h.logger.WithError(err).
			WithField("school_id", schoolID).
			WithField("invoice_id", invoiceID).
			Error("Failed to update invoice status")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update invoice status"})
		return
	}

	resp, err := models.GetInvoiceDetail(h.dbx, schoolID, invoiceID)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse{Message: "Invoice status updated"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteInvoice(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid invoice id"})
		return
	}

	invoice := models.Invoice{BaseModel: models.BaseModel{ID: invoiceID}, SchoolID: schoolID}
	if err := invoice.Get(h.dbx, schoolID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get invoice"})
		return
	}

	if err := invoice.Delete(h.dbx); err != nil {
		if errors.Is(err, models.ErrInvoiceHasPayments) {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "invoice_has_payments",
				Message: "Cannot delete an invoice that has payment records",
			})
			return
		}
		h.logger.WithError(err).
			WithField("school_id", schoolID).
			WithField("invoice_id", invoiceID).
			Error("Failed to delete invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to delete invoice"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Invoice deleted"})
}

func (h *Handler) GrantGrace(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	userID := middleware.GetUserID(c)
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid invoice id"})
		return
	}

	var req models.GrantGraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	graceDate, err := time.Parse(middleware.DateFormat, req.GraceDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid grace date"})
		return
	}

	invoice := models.Invoice{BaseModel: models.BaseModel{ID: invoiceID}, SchoolID: schoolID}
	if err := invoice.Get(h.dbx, schoolID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get invoice")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get invoice"})
		return
	}

	if err := invoice.GrantGrace(h.dbx, userID, graceDate); err != nil {
		h.logger.WithError(err).Error("Failed to grant grace period")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to grant grace period"})
		return
	}

	resp, err := models.GetInvoiceWithItems(h.dbx, schoolID, invoiceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get invoice after grace")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get invoice"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SendReminder(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid invoice id"})
		return
	}
	var req models.SendInvoiceRequest
	_ = c.ShouldBindJSON(&req)
	h.sendInvoiceEmail(c, invoiceID, models.InvoiceSendTypeReminder, req.Email)
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
