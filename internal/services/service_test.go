package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"

	"github.com/school-invoice/backend/internal/config"
	"github.com/school-invoice/backend/internal/database"
	"github.com/school-invoice/backend/internal/handlers"
	"github.com/school-invoice/backend/internal/logger"
	"github.com/school-invoice/backend/internal/middleware"
	"github.com/school-invoice/backend/internal/migrate"
	"github.com/school-invoice/backend/internal/models"
)

func TestRun(t *testing.T) {
	suite.Run(t, new(TestSuiteEducation))
}

type TestSuiteEducation struct {
	TestSuiteData
}

type TestSuiteData struct {
	suite.Suite

	db  *sqlx.DB
	svc *EducationService

	schoolID  uuid.UUID
	userID    uuid.UUID
	role      *models.Role
	sessionID uuid.UUID
	termID    uuid.UUID
	classID   uuid.UUID
	studentID uuid.UUID
	guardianID uuid.UUID
	feeTypeID uuid.UUID
}

func (ts *TestSuiteData) SetupSuite() {
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		ts.T().Skip("DB_USER is not set")
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		ts.T().Skip("DB_NAME is not set")
	}

	migrationsDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	ts.Require().NoError(err)

	ts.db = migrate.BootstrapLocalDatabaseWithPkg(dbUser, dbName, "education_service", migrationsDir)

	log := logger.New("education-test")
	cfg := &config.Config{JWTSecret: "test-secret", Env: "test"}

	ts.svc = &EducationService{
		DBX:    ts.db,
		Redis:  &database.Redis{},
		Config: cfg,
		Log:    log,
	}

	ts.seedBaseData()
}

func (ts *TestSuiteData) SetupTest() {
	gin.SetMode(gin.TestMode)
}

func (ts *TestSuiteData) TearDownSuite() {
	if ts.db != nil {
		_ = ts.db.Close()
	}
	if os.Getenv("DB_USER") != "" && os.Getenv("DB_NAME") != "" {
		migrate.DropDatabaseWithoutExitingWithPkg(os.Getenv("DB_USER"), os.Getenv("DB_NAME"), "education_service")
	}
}

func (ts *TestSuiteData) injectAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeySchoolID, ts.schoolID)
		c.Set(middleware.ContextKeyUserID, ts.userID)
		c.Set(middleware.ContextKeyRoleID, ts.role.ID)
		c.Set(middleware.ContextKeyIsSuperAdmin, true)
		middleware.SetRole(c, ts.role)
		c.Set(middleware.ContextKeyLogger, ts.svc.Log)
		c.Next()
	}
}

func (ts *TestSuiteData) seedBaseData() {
	now := time.Now().UTC()
	ts.schoolID = uuid.New()
	ts.userID = uuid.New()
	roleID := uuid.New()
	ts.sessionID = uuid.New()
	ts.termID = uuid.New()
	ts.classID = uuid.New()

	ts.Require().NoError((&models.School{
		BaseModel: models.BaseModel{ID: ts.schoolID, CreatedAt: now, UpdatedAt: now},
		Name:      "Test School",
		Subdomain: fmt.Sprintf("test-%s", ts.schoolID.String()[:8]),
		Phone:     "+2348012345678",
		Email:     fmt.Sprintf("school-%s@test.com", ts.schoolID.String()[:8]),
		Address:   "123 Test Street",
		IsActive:  true,
	}).Create(ts.db))

	ts.role = &models.Role{
		BaseModel:    models.BaseModel{ID: roleID, CreatedAt: now, UpdatedAt: now},
		SchoolID:     ts.schoolID,
		Name:         "Super Admin",
		Description:  "Test role",
		Permissions:  models.GetSuperAdminPermissions(),
		IsSuperAdmin: true,
	}
	ts.Require().NoError(ts.role.Create(ts.db))

	ts.Require().NoError((&models.User{
		BaseModel:    models.BaseModel{ID: ts.userID, CreatedAt: now, UpdatedAt: now},
		SchoolID:     ts.schoolID,
		RoleID:       roleID,
		Email:        fmt.Sprintf("admin-%s@test.com", ts.userID.String()[:8]),
		PasswordHash: "hashed-password",
		FirstName:    "Test",
		LastName:     "Admin",
		Phone:        "+2348012345679",
		IsActive:     func(v bool) *bool { return &v }(true),
	}).Create(ts.db))

	start := now.AddDate(0, -1, 0)
	end := now.AddDate(0, 8, 0)
	isCurrent := true
	ts.Require().NoError((&models.AcademicSession{
		BaseModel: models.BaseModel{ID: ts.sessionID, CreatedAt: now, UpdatedAt: now},
		SchoolID:  ts.schoolID,
		Name:      "2025/2026",
		StartDate: &start,
		EndDate:   &end,
		IsCurrent: &isCurrent,
	}).Create(ts.db))

	termStart := now.AddDate(0, -1, 0)
	termEnd := now.AddDate(0, 2, 0)
	ts.Require().NoError((&models.Term{
		BaseModel: models.BaseModel{ID: ts.termID, CreatedAt: now, UpdatedAt: now},
		SchoolID:  ts.schoolID,
		SessionID: ts.sessionID,
		Name:      models.FirstTerm,
		SortOrder: 1,
		StartDate: &termStart,
		EndDate:   &termEnd,
		IsCurrent: &isCurrent,
	}).Create(ts.db))

	ts.Require().NoError((&models.Class{
		BaseModel: models.BaseModel{ID: ts.classID, CreatedAt: now, UpdatedAt: now},
		SchoolID:  ts.schoolID,
		Name:      "JSS1",
		Section:   "A",
		SortOrder: 1,
		IsActive:  true,
	}).Create(ts.db))
}

func (ts *TestSuiteData) createStudent() uuid.UUID {
	dob := time.Date(2012, 5, 15, 0, 0, 0, 0, time.UTC)
	student := models.Student{
		FirstName:   "John",
		MiddleName:  "Michael",
		LastName:    "Doe",
		Gender:      models.GenderMale,
		DateOfBirth: &dob,
		NIN:         fmt.Sprintf("NIN%s", uuid.NewString()[:8]),
	}
	ts.Require().NoError(student.Create(ts.db))

	admission := models.StudentAdmission{
		StudentID:     student.ID,
		SchoolID:      ts.schoolID,
		AdmissionNo:   fmt.Sprintf("ADM%s", uuid.NewString()[:8]),
		AdmissionDate: time.Now().UTC(),
	}
	ts.Require().NoError(admission.Create(ts.db))

	enrollment := models.StudentEnrollment{
		BaseModel:  models.NewBaseModel(),
		SchoolID:   ts.schoolID,
		StudentID:  student.ID,
		ClassID:    ts.classID,
		TermID:     ts.termID,
		Status:     models.EnrollmentActive,
		EnrolledAt: time.Now().UTC(),
	}
	ts.Require().NoError(enrollment.Create(ts.db))

	return student.ID
}

func (ts *TestSuiteData) createGuardian() uuid.UUID {
	guardian := models.Guardian{
		FirstName: "Jane",
		LastName:  "Doe",
		Phone:     fmt.Sprintf("+23480%08d", time.Now().UnixNano()%100000000),
		Email:     fmt.Sprintf("guardian-%s@test.com", uuid.NewString()[:8]),
		Address:   "456 Parent Street",
	}
	ts.Require().NoError(guardian.Create(ts.db))
	return guardian.ID
}

func (ts *TestSuiteData) createFeeType() uuid.UUID {
	feeType := models.FeeType{
		BaseModel:     models.NewBaseModel(),
		SchoolID:      ts.schoolID,
		Name:          "Tuition Fee",
		Description:   "Term tuition",
		DefaultAmount: decimal.NewFromFloat(30000),
		Category:      models.FeeCategoryAcademic,
		Frequency:     models.FeeFrequencyPerTerm,
		IsOptional:    false,
		IsActive:      true,
	}
	ts.Require().NoError(feeType.Create(ts.db))
	return feeType.ID
}

func (ts *TestSuiteData) newRouter(injections []gin.HandlerFunc, register RouteRegister) *gin.Engine {
	if len(injections) == 0 {
		injections = []gin.HandlerFunc{ts.injectAuth()}
	}
	return ts.svc.TestingRouter(injections, register)
}

func (ts *TestSuiteData) doRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		ts.Require().NoError(err)
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func (ts *TestSuiteData) TestListAndUpdateClass() {
	router := ts.newRouter(nil, func(protected *gin.RouterGroup, h *handlers.Handler) {
		ts.svc.setupClassRoutes(protected, h)
	})

	w := ts.doRequest(router, http.MethodGet, "/api/v1/classes", nil)
	ts.Equal(http.StatusOK, w.Code)

	createBody := map[string]interface{}{
		"name":       "JSS2",
		"section":    "B",
		"sort_order": 2,
	}
	w = ts.doRequest(router, http.MethodPost, "/api/v1/classes", createBody)
	ts.Equal(http.StatusCreated, w.Code)

	var created models.Class
	ts.Require().NoError(json.Unmarshal(w.Body.Bytes(), &created))

	updateBody := map[string]interface{}{
		"name":       "JSS2",
		"section":    "C",
		"sort_order": 3,
		"is_active":  true,
	}
	w = ts.doRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/classes/%s", created.ID), updateBody)
	ts.Equal(http.StatusOK, w.Code)

	var updated models.ClassResponse
	ts.Require().NoError(json.Unmarshal(w.Body.Bytes(), &updated))
	ts.Equal("C", updated.Section)
}

func (ts *TestSuiteData) TestGetClassStudents() {
	ts.studentID = ts.createStudent()

	router := ts.newRouter(nil, func(protected *gin.RouterGroup, h *handlers.Handler) {
		ts.svc.setupClassRoutes(protected, h)
	})

	w := ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/classes/%s/students", ts.classID), nil)
	ts.Equal(http.StatusOK, w.Code)

	var students []models.Student
	ts.Require().NoError(json.Unmarshal(w.Body.Bytes(), &students))
	ts.NotEmpty(students)
}

func (ts *TestSuiteData) TestEnrollmentEndpoints() {
	ts.studentID = ts.createStudent()

	router := ts.newRouter(nil, func(protected *gin.RouterGroup, h *handlers.Handler) {
		ts.svc.setupEnrollmentRoutes(protected, h)
	})

	w := ts.doRequest(router, http.MethodGet, "/api/v1/enrollments", nil)
	ts.Equal(http.StatusOK, w.Code)

	newClass := models.Class{
		BaseModel: models.NewBaseModel(),
		SchoolID:  ts.schoolID,
		Name:      "JSS2",
		Section:   "A",
		SortOrder: 2,
		IsActive:  true,
	}
	ts.Require().NoError(newClass.Create(ts.db))

	createBody := map[string]interface{}{
		"student_id": ts.studentID,
		"class_id":   newClass.ID,
		"term_id":    ts.termID,
	}
	w = ts.doRequest(router, http.MethodPost, "/api/v1/enrollments", createBody)
	ts.Equal(http.StatusConflict, w.Code)

	bulkBody := map[string]interface{}{
		"from_class_id": ts.classID,
		"to_class_id":   newClass.ID,
		"term_id":       ts.termID,
	}
	w = ts.doRequest(router, http.MethodPost, "/api/v1/enrollments/bulk", bulkBody)
	ts.Equal(http.StatusCreated, w.Code)

	var enrollments []models.EnrollmentResponse
	w = ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/enrollments?class_id=%s", newClass.ID), nil)
	ts.Equal(http.StatusOK, w.Code)
	ts.Require().NoError(json.Unmarshal(w.Body.Bytes(), &enrollments))
	ts.NotEmpty(enrollments)

	statusBody := map[string]interface{}{"status": "promoted"}
	w = ts.doRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/enrollments/%s/status", enrollments[0].ID), statusBody)
	ts.Equal(http.StatusOK, w.Code)
}

func (ts *TestSuiteData) TestGuardianEndpoints() {
	ts.studentID = ts.createStudent()
	ts.guardianID = ts.createGuardian()

	router := ts.newRouter(nil, func(protected *gin.RouterGroup, h *handlers.Handler) {
		ts.svc.setupStudentRoutes(protected, h)
		ts.svc.setupGuardianRoutes(protected, h)
	})

	linkBody := map[string]interface{}{
		"guardian_id":            ts.guardianID,
		"relationship":           "mother",
		"is_primary":             true,
		"receives_notifications": true,
	}
	w := ts.doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/students/%s/guardians", ts.studentID), linkBody)
	ts.Equal(http.StatusCreated, w.Code)

	w = ts.doRequest(router, http.MethodGet, "/api/v1/guardians", nil)
	ts.Equal(http.StatusOK, w.Code)

	w = ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/guardians/%s", ts.guardianID), nil)
	ts.Equal(http.StatusOK, w.Code)

	updateBody := map[string]interface{}{
		"first_name": "Janet",
		"last_name":  "Doe",
	}
	w = ts.doRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/guardians/%s", ts.guardianID), updateBody)
	ts.Equal(http.StatusOK, w.Code)

	w = ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/guardians/%s/invoices", ts.guardianID), nil)
	ts.Equal(http.StatusOK, w.Code)
}

func (ts *TestSuiteData) TestStudentEndpoints() {
	ts.studentID = ts.createStudent()
	ts.guardianID = ts.createGuardian()

	router := ts.newRouter(nil, func(protected *gin.RouterGroup, h *handlers.Handler) {
		ts.svc.setupStudentRoutes(protected, h)
	})

	w := ts.doRequest(router, http.MethodGet, "/api/v1/students", nil)
	ts.Equal(http.StatusOK, w.Code)

	w = ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/students/%s/enrollments", ts.studentID), nil)
	ts.Equal(http.StatusOK, w.Code)

	linkBody := map[string]interface{}{
		"guardian_id":            ts.guardianID,
		"relationship":           "father",
		"is_primary":             true,
		"receives_notifications": true,
	}
	w = ts.doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/students/%s/guardians", ts.studentID), linkBody)
	ts.Equal(http.StatusCreated, w.Code)

	w = ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/students/%s/guardians", ts.studentID), nil)
	ts.Equal(http.StatusOK, w.Code)

	// One father per student
	secondGuardian := ts.createGuardian()
	dupFather := map[string]interface{}{
		"guardian_id":            secondGuardian,
		"relationship":           "father",
		"is_primary":             false,
		"receives_notifications": true,
	}
	w = ts.doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/students/%s/guardians", ts.studentID), dupFather)
	ts.Equal(http.StatusConflict, w.Code)

	// Same guardian can be linked to another student
	siblingID := ts.createStudent()
	siblingLink := map[string]interface{}{
		"guardian_id":            ts.guardianID,
		"relationship":           "father",
		"is_primary":             true,
		"receives_notifications": true,
	}
	w = ts.doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/students/%s/guardians", siblingID), siblingLink)
	ts.Equal(http.StatusCreated, w.Code)

	w = ts.doRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/students/%s/guardians/%s", siblingID, ts.guardianID), nil)
	ts.Equal(http.StatusOK, w.Code)
}

func (ts *TestSuiteData) TestFeeTypeEndpoints() {
	ts.feeTypeID = ts.createFeeType()

	router := ts.newRouter(nil, func(protected *gin.RouterGroup, h *handlers.Handler) {
		ts.svc.setupFeeTypeRoutes(protected, h)
	})

	w := ts.doRequest(router, http.MethodGet, "/api/v1/fee-types", nil)
	ts.Equal(http.StatusOK, w.Code)

	createBody := map[string]interface{}{
		"name":           "Lab Fee",
		"description":    "Science lab",
		"default_amount": 5000,
		"category":       "academic",
		"frequency":      "per_term",
		"is_optional":    false,
	}
	w = ts.doRequest(router, http.MethodPost, "/api/v1/fee-types", createBody)
	ts.Equal(http.StatusCreated, w.Code)

	var created models.FeeTypeResponse
	ts.Require().NoError(json.Unmarshal(w.Body.Bytes(), &created))

	updateBody := map[string]interface{}{
		"name":          "Updated Lab Fee",
		"default_amount": 6000,
	}
	w = ts.doRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/fee-types/%s", created.ID), updateBody)
	ts.Equal(http.StatusOK, w.Code)

	amountsBody := map[string]interface{}{
		"amounts": []map[string]interface{}{
			{"class_id": ts.classID, "amount": 5500},
		},
	}
	w = ts.doRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/fee-types/%s/amounts", created.ID), amountsBody)
	ts.Equal(http.StatusOK, w.Code)

	w = ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/fee-types/%s/amounts", created.ID), nil)
	ts.Equal(http.StatusOK, w.Code)
}

func (ts *TestSuiteData) TestInvoiceEndpoints() {
	ts.studentID = ts.createStudent()
	ts.feeTypeID = ts.createFeeType()
	ts.guardianID = ts.createGuardian()

	router := ts.newRouter(nil, func(protected *gin.RouterGroup, h *handlers.Handler) {
		ts.svc.setupStudentRoutes(protected, h)
		ts.svc.setupGuardianRoutes(protected, h)
		ts.svc.setupClassRoutes(protected, h)
		ts.svc.setupInvoiceRoutes(protected, h)
	})

	linkBody := map[string]interface{}{
		"guardian_id":            ts.guardianID,
		"relationship":           "mother",
		"is_primary":             true,
		"receives_notifications": true,
	}
	w := ts.doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/students/%s/guardians", ts.studentID), linkBody)
	ts.Equal(http.StatusCreated, w.Code)

	createBody := map[string]interface{}{
		"student_id":   ts.studentID,
		"fee_type_ids": []uuid.UUID{ts.feeTypeID},
		"due_date":     time.Now().AddDate(0, 0, 14).Format("2006-01-02"),
	}
	w = ts.doRequest(router, http.MethodPost, "/api/v1/invoices", createBody)
	ts.Equal(http.StatusCreated, w.Code)

	var invoice models.InvoiceResponse
	ts.Require().NoError(json.Unmarshal(w.Body.Bytes(), &invoice))
	ts.NotEmpty(invoice.InvoiceNo)
	ts.Greater(invoice.TotalAmount, float64(0))

	w = ts.doRequest(router, http.MethodGet, "/api/v1/invoices", nil)
	ts.Equal(http.StatusOK, w.Code)

	w = ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/invoices/%s", invoice.ID), nil)
	ts.Equal(http.StatusOK, w.Code)

	w = ts.doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/invoices/%s/send", invoice.ID), nil)
	ts.Equal(http.StatusOK, w.Code)

	graceBody := map[string]interface{}{
		"grace_date": time.Now().AddDate(0, 0, 21).Format("2006-01-02"),
		"reason":     "Family hardship",
	}
	w = ts.doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/invoices/%s/grace", invoice.ID), graceBody)
	ts.Equal(http.StatusOK, w.Code)

	w = ts.doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/invoices/%s/remind", invoice.ID), nil)
	ts.Equal(http.StatusOK, w.Code)

	w = ts.doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/guardians/%s/invoices", ts.guardianID), nil)
	ts.Equal(http.StatusOK, w.Code)

	bulkBody := map[string]interface{}{
		"class_id":     ts.classID,
		"fee_type_ids": []uuid.UUID{ts.feeTypeID},
		"due_date":     time.Now().AddDate(0, 0, 14).Format("2006-01-02"),
	}
	w = ts.doRequest(router, http.MethodPost, "/api/v1/invoices/bulk", bulkBody)
	ts.Equal(http.StatusCreated, w.Code)
}
