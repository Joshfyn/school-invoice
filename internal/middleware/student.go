package middleware

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-invoice/backend/internal/dto"
	"github.com/school-invoice/backend/internal/logger"
)

func ValidateCreateStudentRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Set(ReqBodyCreateStudent, req)
	c.Next()
}

func ValidateUpdateStudentRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.UpdateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.
			WithError(err).
			Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	studentID := c.Param("id")
	if studentID == "" {
		logger.Error("Student ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Student ID is required"})
		return
	}
	req.StudentID = uuid.MustParse(studentID)
	if !req.DateOfBirth.IsZero() {
		logger.Error("Date of birth is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Date of birth is required"})
		return
	}

	c.Set(ReqBodyUpdateStudent, req)
	c.Next()
}

func ValidateGetSingleStudentRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.GetSingleStudentRequest
	studentID := c.Param("id")
	if studentID == "" {
		logger.Error("Student ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Student ID is required"})
		return
	}
	req.StudentID = uuid.MustParse(studentID)
	c.Set(ReqBodyGetSingleStudent, req)
	c.Next()
}

func ValidateCreateStudentAdmissionRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.CreateStudentAdmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithError(err).Error("Failed to bind JSON")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.SchoolID = GetSchoolID(c)

	admissionDate, err := time.Parse(DateFormat, req.AdmissionDate)
	if err != nil {
		logger.
			WithError(err).
			WithField("school_id", req.SchoolID).
			WithField("student_id", req.StudentID).
			WithField("admission_date", req.AdmissionDate).
			Error("Invalid admission date")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid admission date"})
		return
	}
	req.AdmissionDate = admissionDate.Format(DateFormat)

	req.AdmissionNo = generate11DigitAdmissionNo(req.SchoolID.String(), req.StudentID.String())

	c.Set(ReqBodyCreateStudentAdmission, req)
	c.Next()

}

func ValidateDeleteStudentAdmissionRequest(c *gin.Context) {
	logger := c.MustGet(ContextKeyLogger).(*logger.Logger)
	var req dto.DeleteStudentAdmissionRequest
	studentID := c.Param("id")
	if studentID == "" {
		logger.Error("Student ID is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Student ID is required"})
		return
	}
	req.StudentID = uuid.MustParse(studentID)
	req.SchoolID = GetSchoolID(c)
	c.Set(ReqBodyDeleteStudentAdmission, req)
	c.Next()
}

func generate11DigitAdmissionNo(schoolID string, studentID string) string {
	// 1. Combine inputs into a unique string
	// Using separators (|) prevents "John" + "Smith" from clashing with "Joh" + "nSmith"
	input := fmt.Sprintf("%s|%s", schoolID, studentID)

	// 2. Create a new FNV-1a 64-bit hash
	h := fnv.New64a()
	h.Write([]byte(input))
	hashValue := h.Sum64()

	// 3. Map to 11 digits (Range: 10,000,000,000 to 99,999,999,999)
	// We use 90 billion as the range and add 10 billion to ensure it's always 11 digits
	const min11Digit = 10000000000
	const max11Digit = 99999999999

	finalID := (hashValue % (max11Digit - min11Digit + 1)) + min11Digit

	return fmt.Sprintf("%d", finalID)
}
