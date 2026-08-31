package services

import (
	"github.com/gin-gonic/gin"
	"github.com/school-invoice/backend/internal/handlers"
	"github.com/school-invoice/backend/internal/middleware"
)

// SetupRouter creates and configures the Gin router with all routes
func (svc *EducationService) SetupRouter() *gin.Engine {
	router := gin.New()

	// Add middleware
	router.Use(middleware.AddLogger(svc.Log))
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	// Initialize handlers with service dependencies
	h := handlers.New(svc.DBX, svc.Redis, svc.Config, svc.Log)

	// Setup all routes
	svc.setupRoutes(router, h)

	return router
}

// RouteRegister wires protected routes onto a test router.
type RouteRegister func(protected *gin.RouterGroup, h *handlers.Handler)

// TestingRouter configures a minimal router for integration tests.
// Callers provide auth/test middleware via injections and register only the routes under test.
func (svc *EducationService) TestingRouter(injections []gin.HandlerFunc, register RouteRegister) *gin.Engine {
	router := gin.New()
	router.Use(middleware.AddLogger(svc.Log))
	router.Use(gin.Recovery())

	h := handlers.New(svc.DBX, svc.Redis, svc.Config, svc.Log)

	api := router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(injections...)
	if register != nil {
		register(protected, h)
	}

	return router
}

// setupRoutes configures all API routes
func (svc *EducationService) setupRoutes(router *gin.Engine, h *handlers.Handler) {
	// Health check
	router.GET("/health", h.Health)

	// API v1
	api := router.Group("/api/v1")
	{
		// Public routes (no auth required)
		svc.setupAuthRoutes(api, h)
		svc.setupPublicRoutes(api, h)

		// Protected routes (auth required)
		protected := api.Group("")
		protected.Use(middleware.Auth(svc.Config.JWTSecret, svc.Redis, svc.DBX))
		{
			svc.setupSchoolRoutes(protected, h)
			svc.setupRoleRoutes(protected, h)
			svc.setupUserRoutes(protected, h)
			svc.setupSessionRoutes(protected, h)
			svc.setupTermRoutes(protected, h)
			svc.setupClassRoutes(protected, h)
			svc.setupEnrollmentRoutes(protected, h)
			svc.setupGuardianRoutes(protected, h)
			svc.setupStudentRoutes(protected, h)
			svc.setupFeeTypeRoutes(protected, h)
			svc.setupInvoiceRoutes(protected, h)
			svc.setupPaymentRoutes(protected, h)
			svc.setupReportRoutes(protected, h)
		}
	}
}

// setupAuthRoutes configures authentication routes
func (svc *EducationService) setupAuthRoutes(api *gin.RouterGroup, h *handlers.Handler) {
	auth := api.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", middleware.ValidateResetToken, h.ResetPassword)
	}

	api.GET("/auth/me", middleware.Auth(svc.Config.JWTSecret, svc.Redis, svc.DBX), h.GetMe)
}

// setupPublicRoutes configures public routes (no auth required)
func (svc *EducationService) setupPublicRoutes(api *gin.RouterGroup, h *handlers.Handler) {
	// Public invoice view (for parents)
	api.GET("/invoice/:id", h.GetPublicInvoice)

	// Payment webhook (Paystack)
	api.POST("/payments/webhook", h.PaymentWebhook)

	// Guardian portal (access is granted by the signed token in invoice/reminder emails)
	portal := api.Group("/portal/access/:token")
	{
		portal.GET("", h.GetPortalSession)
		portal.GET("/invoices/:id", h.GetPortalInvoice)
		portal.GET("/invoices/:id/pdf", h.GetPortalInvoicePDF)
		portal.POST("/invoices/:id/pay", h.InitializePortalPayment)
		portal.POST("/invoices/:id/grace-request", h.CreatePortalGraceRequest)
	}
}

// setupSchoolRoutes configures school profile routes
func (svc *EducationService) setupSchoolRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	schools := protected.Group("/schools")
	{
		schools.GET("/profile", h.GetSchoolProfile)
		schools.PUT("/profile", middleware.ValidateSchoolProfileUpdate, middleware.RequirePermission("settings", "manage"), h.UpdateSchoolProfile)
	}
}

// setupRoleRoutes configures role management routes
func (svc *EducationService) setupRoleRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	roles := protected.Group("/roles")
	{
		roles.GET("", middleware.RequirePermission("roles", "manage"), h.ListRoles)
		roles.POST("", middleware.ValidateCreateRoleRequest, middleware.RequirePermission("roles", "manage"), h.CreateRole)
		roles.GET("/:id", middleware.RequirePermission("roles", "manage"), h.GetRole)
		roles.PUT("/:id", middleware.ValidateUpdateRoleRequest, middleware.RequirePermission("roles", "manage"), h.UpdateRole)
		roles.DELETE("/:id", middleware.ValidateDeleteRoleRequest, middleware.RequirePermission("roles", "manage"), h.DeleteRole)
	}
}

// setupUserRoutes configures user management routes
func (svc *EducationService) setupUserRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	users := protected.Group("/users")
	{
		users.GET("", middleware.RequirePermission("users", "read"), h.ListUsers)
		users.POST("", middleware.RequirePermission("users", "create"), middleware.ValidateCreateUserRequest, h.CreateUser)
		users.GET("/:id", middleware.ValidateGetSingleUserRequest, middleware.RequirePermission("users", "read"), h.GetSingleUser)
		users.PUT("/:id", middleware.ValidateUpdateUserRequest, middleware.RequirePermission("users", "update"), h.UpdateUser)
		users.PUT("/:id/role", middleware.ValidateUpdateUserRoleRequest, middleware.RequireAnyPermission(
			middleware.PermissionCheck{Resource: "users", Action: "update"},
			middleware.PermissionCheck{Resource: "roles", Action: "manage"},
		), h.UpdateUserRole)
		users.PUT("/:id/status", middleware.ValidateUpdateUserStatusRequest, middleware.RequirePermission("users", "update"), h.UpdateUserStatus)
		//users.GET("/:id/class-access", middleware.RequirePermission("users", "read"), h.GetUserClassAccess)
		//users.PUT("/:id/class-access", middleware.RequirePermission("classes", "manage"), h.SetUserClassAccess)
	}
}

// setupSessionRoutes configures academic session routes
func (svc *EducationService) setupSessionRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	sessions := protected.Group("/sessions")
	{
		sessions.GET("", middleware.RequireAnyPermission(
			middleware.PermissionCheck{Resource: "sessions", Action: "manage"},
			middleware.PermissionCheck{Resource: "students", Action: "read"},
			middleware.PermissionCheck{Resource: "students", Action: "create"},
		), h.ListSessions)
		sessions.POST("", middleware.ValidateCreateSessionRequest, middleware.RequirePermission("sessions", "manage"), h.CreateSession)
		sessions.PUT("/:id", middleware.ValidateUpdateSessionRequest, middleware.RequirePermission("sessions", "manage"), h.UpdateSession)
		sessions.PUT("/:id/current", middleware.ValidateSetCurrentSessionRequest, middleware.RequirePermission("sessions", "manage"), h.SetCurrentSession)
	}
}

// setupTermRoutes configures term routes
func (svc *EducationService) setupTermRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	terms := protected.Group("/terms")
	{
		terms.GET("", middleware.RequireAnyPermission(
			middleware.PermissionCheck{Resource: "sessions", Action: "manage"},
			middleware.PermissionCheck{Resource: "students", Action: "read"},
			middleware.PermissionCheck{Resource: "students", Action: "create"},
		), h.ListTerms)
		terms.POST("", middleware.ValidateCreateTermRequest, middleware.RequirePermission("sessions", "manage"), h.CreateTerm)
		terms.PUT("/:id", middleware.ValidateUpdateTermRequest, middleware.RequirePermission("sessions", "manage"), h.UpdateTerm)
		terms.PUT("/:id/current", middleware.ValidateSetCurrentTermRequest, middleware.RequirePermission("sessions", "manage"), h.SetCurrentTerm)
	}
}

// setupClassRoutes configures class routes
func (svc *EducationService) setupClassRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	classes := protected.Group("/classes")
	{
		classes.GET("", middleware.RequireAnyPermission(
			middleware.PermissionCheck{Resource: "classes", Action: "manage"},
			middleware.PermissionCheck{Resource: "students", Action: "read"},
			middleware.PermissionCheck{Resource: "students", Action: "create"},
		), h.ListClasses)
		classes.POST("", middleware.ValidateCreateClassRequest, middleware.RequirePermission("classes", "manage"), h.CreateClass)
		classes.PUT("/:id", middleware.ValidateUpdateClassRequest, middleware.RequirePermission("classes", "manage"), h.UpdateClass)
		classes.GET("/:id/students", middleware.RequirePermission("students", "read"), h.GetClassStudents)
	}
}

// setupEnrollmentRoutes configures enrollment routes
func (svc *EducationService) setupEnrollmentRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	enrollments := protected.Group("/enrollments")
	{
		enrollments.GET("", middleware.RequirePermission("students", "read"), h.ListEnrollments)
		enrollments.POST("", middleware.RequirePermission("students", "create"), h.CreateEnrollment)
		enrollments.POST("/bulk", middleware.RequirePermission("students", "create"), h.BulkEnrollment)
		enrollments.PUT("/:id/status", middleware.RequirePermission("students", "update"), h.UpdateEnrollmentStatus)
	}
}

// setupGuardianRoutes configures guardian routes
func (svc *EducationService) setupGuardianRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	guardians := protected.Group("/guardians")
	{
		guardians.GET("", middleware.RequirePermission("guardians", "read"), h.ListGuardians)
		guardians.POST("", middleware.RequirePermission("guardians", "create"), middleware.ValidateCreateGuardianRequest, h.CreateGuardian)
		guardians.GET("/:id", middleware.RequirePermission("guardians", "read"), h.GetGuardian)
		guardians.PUT("/:id", middleware.RequirePermission("guardians", "update"), middleware.ValidateUpdateGuardianRequest, h.UpdateGuardian)
		guardians.GET("/:id/invoices", middleware.RequirePermission("invoices", "read"), h.GetGuardianInvoices)
	}
}

// setupStudentRoutes configures student routes
func (svc *EducationService) setupStudentRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	students := protected.Group("/students")
	{
		students.GET("", middleware.RequirePermission("students", "read"), h.ListStudents)
		students.POST("", middleware.ValidateCreateStudentRequest, middleware.RequirePermission("students", "create"), h.CreateStudent)
		students.POST("/bulk", middleware.RequirePermission("students", "create"), h.BulkCreateStudents)
		students.POST("/admission", middleware.ValidateCreateStudentAdmissionRequest, middleware.RequirePermission("students", "create"), h.CreateStudentAdmission)
		students.GET("/:id", middleware.ValidateGetSingleStudentRequest, middleware.RequirePermission("students", "read"), h.GetStudent)
		students.PUT("/:id", middleware.ValidateUpdateStudentRequest, middleware.RequirePermission("students", "update"), h.UpdateStudent)
		students.DELETE("/:id/admission", middleware.ValidateDeleteStudentAdmissionRequest, middleware.RequirePermission("students", "delete"), h.DeleteStudentAdmission)
		students.GET("/:id/enrollments", middleware.RequirePermission("students", "read"), h.GetStudentEnrollments)
		students.GET("/:id/guardians", middleware.RequirePermission("guardians", "read"), h.GetStudentGuardians)
		students.POST("/:id/guardians", middleware.RequirePermission("guardians", "create"), h.LinkStudentGuardian)
		students.DELETE("/:id/guardians/:guardianId", middleware.RequirePermission("guardians", "update"), h.UnlinkStudentGuardian)
	}
}

// setupFeeTypeRoutes configures fee type routes
func (svc *EducationService) setupFeeTypeRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	feeTypes := protected.Group("/fee-types")
	{
		feeTypes.GET("", middleware.RequirePermission("fee_types", "read"), h.ListFeeTypes)
		feeTypes.POST("", middleware.RequirePermission("fee_types", "create"), h.CreateFeeType)
		feeTypes.PUT("/:id", middleware.RequirePermission("fee_types", "update"), h.UpdateFeeType)
		feeTypes.DELETE("/:id", middleware.RequirePermission("fee_types", "delete"), h.DeleteFeeType)
		feeTypes.GET("/:id/amounts", middleware.RequirePermission("fee_types", "read"), h.GetFeeTypeAmounts)
		feeTypes.PUT("/:id/amounts", middleware.RequirePermission("fee_types", "set_amounts"), h.SetFeeTypeAmounts)
	}

	feeCategories := protected.Group("/fee-categories")
	{
		feeCategories.GET("", middleware.RequirePermission("fee_types", "read"), h.ListFeeCategories)
		feeCategories.POST("", middleware.RequirePermission("settings", "manage"), h.CreateFeeCategory)
		feeCategories.DELETE("/:id", middleware.RequirePermission("settings", "manage"), h.DeleteFeeCategory)
	}
}

// setupInvoiceRoutes configures invoice routes
func (svc *EducationService) setupInvoiceRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	invoices := protected.Group("/invoices")
	{
		invoices.GET("", middleware.RequirePermission("invoices", "read"), h.ListInvoices)
		invoices.POST("", middleware.RequirePermission("invoices", "create"), h.CreateInvoice)
		invoices.POST("/bulk", middleware.RequirePermission("invoices", "bulk_create"), h.BulkCreateInvoices)
		invoices.GET("/:id", middleware.RequirePermission("invoices", "read"), h.GetInvoice)
		invoices.PUT("/:id/status", middleware.RequireSuperAdmin(), h.UpdateInvoiceStatus)
		invoices.DELETE("/:id", middleware.RequireSuperAdmin(), h.DeleteInvoice)
		invoices.POST("/:id/send", middleware.RequirePermission("invoices", "send"), h.SendInvoice)
		invoices.POST("/:id/grace", middleware.RequirePermission("invoices", "grant_grace"), h.GrantGrace)
		invoices.POST("/:id/remind", middleware.RequirePermission("invoices", "send"), h.SendReminder)
	}
}

// setupPaymentRoutes configures payment routes
func (svc *EducationService) setupPaymentRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	payments := protected.Group("/payments")
	{
		payments.POST("/initialize", h.InitializePayment)
		payments.POST("/verify-bank", middleware.RequirePermission("payments", "verify"), h.VerifyBankPayment)
		payments.GET("/receipt/:id", middleware.RequirePermission("payments", "read"), h.GetReceipt)
	}
}

// setupReportRoutes configures report routes
func (svc *EducationService) setupReportRoutes(protected *gin.RouterGroup, h *handlers.Handler) {
	reports := protected.Group("/reports")
	{
		reports.GET("/summary", middleware.RequirePermission("reports", "view"), h.GetDashboardSummary)
		reports.GET("/outstanding", middleware.RequirePermission("reports", "view"), h.GetOutstandingReport)
		reports.GET("/collections", middleware.RequirePermission("reports", "view"), h.GetCollectionReport)
	}
}
