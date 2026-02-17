package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Permission represents the available permissions in the system
type Permission string

const (
	// Student permissions
	PermStudentsCreate Permission = "students:create"
	PermStudentsRead   Permission = "students:read"
	PermStudentsUpdate Permission = "students:update"
	PermStudentsDelete Permission = "students:delete"

	// Guardian permissions
	PermGuardiansCreate Permission = "guardians:create"
	PermGuardiansRead   Permission = "guardians:read"
	PermGuardiansUpdate Permission = "guardians:update"
	PermGuardiansDelete Permission = "guardians:delete"

	// Invoice permissions
	PermInvoicesCreate     Permission = "invoices:create"
	PermInvoicesRead       Permission = "invoices:read"
	PermInvoicesUpdate     Permission = "invoices:update"
	PermInvoicesSend       Permission = "invoices:send"
	PermInvoicesGrantGrace Permission = "invoices:grant_grace"
	PermInvoicesBulkCreate Permission = "invoices:bulk_create"

	// Payment permissions
	PermPaymentsRead       Permission = "payments:read"
	PermPaymentsVerify     Permission = "payments:verify"
	PermPaymentsRecordBank Permission = "payments:record_bank"

	// Report permissions
	PermReportsView   Permission = "reports:view"
	PermReportsExport Permission = "reports:export"

	// User permissions
	PermUsersCreate Permission = "users:create"
	PermUsersRead   Permission = "users:read"
	PermUsersUpdate Permission = "users:update"
	PermUsersDelete Permission = "users:delete"

	// Role permissions
	PermRolesManage Permission = "roles:manage"

	// Settings permissions
	PermSettingsManage Permission = "settings:manage"

	// Session/Term permissions
	PermSessionsManage Permission = "sessions:manage"

	// Class permissions
	PermClassesManage Permission = "classes:manage"

	// Fee type permissions
	PermFeeTypesCreate     Permission = "fee_types:create"
	PermFeeTypesRead       Permission = "fee_types:read"
	PermFeeTypesUpdate     Permission = "fee_types:update"
	PermFeeTypesDelete     Permission = "fee_types:delete"
	PermFeeTypesSetAmounts Permission = "fee_types:set_amounts"
)

// Permissions is a map of resource to allowed actions
type Permissions map[string][]string

// Role represents a role in the system
type Role struct {
	BaseModel
	SchoolID     uuid.UUID   `json:"school_id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Permissions  Permissions `json:"permissions"`
	IsSuperAdmin bool        `json:"is_super_admin"`
}

// CreateRoleRequest is the request body for creating a role
type CreateRoleRequest struct {
	Name        string      `json:"name" binding:"required,min=2,max=100"`
	Description string      `json:"description" binding:"max=500"`
	Permissions Permissions `json:"permissions" binding:"required"`
}

// UpdateRoleRequest is the request body for updating a role
type UpdateRoleRequest struct {
	Name        *string      `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Description *string      `json:"description,omitempty" binding:"omitempty,max=500"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

// RoleResponse is the response for role data
type RoleResponse struct {
	ID           uuid.UUID   `json:"id"`
	SchoolID     uuid.UUID   `json:"school_id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Permissions  Permissions `json:"permissions"`
	IsSuperAdmin bool        `json:"is_super_admin"`
}

func (r *Role) ToResponse() RoleResponse {
	return RoleResponse{
		ID:           r.ID,
		SchoolID:     r.SchoolID,
		Name:         r.Name,
		Description:  r.Description,
		Permissions:  r.Permissions,
		IsSuperAdmin: r.IsSuperAdmin,
	}
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(resource string, action string) bool {
	if r.IsSuperAdmin {
		return true
	}

	actions, ok := r.Permissions[resource]
	if !ok {
		return false
	}

	for _, a := range actions {
		if a == action || a == "*" {
			return true
		}
	}
	return false
}

// PermissionsToJSON converts Permissions to JSON bytes
func (p Permissions) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// PermissionsFromJSON converts JSON bytes to Permissions
func PermissionsFromJSON(data []byte) (Permissions, error) {
	var p Permissions
	err := json.Unmarshal(data, &p)
	return p, err
}

// GetSuperAdminPermissions returns all permissions for super admin
func GetSuperAdminPermissions() Permissions {
	return Permissions{
		"students":  []string{"create", "read", "update", "delete"},
		"guardians": []string{"create", "read", "update", "delete"},
		"invoices":  []string{"create", "read", "update", "send", "grant_grace", "bulk_create"},
		"payments":  []string{"read", "verify", "record_bank"},
		"reports":   []string{"view", "export"},
		"users":     []string{"create", "read", "update", "delete"},
		"roles":     []string{"manage"},
		"settings":  []string{"manage"},
		"sessions":  []string{"manage"},
		"classes":   []string{"manage"},
		"fee_types": []string{"create", "read", "update", "delete", "set_amounts"},
	}
}

func (r *Role) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		INSERT INTO roles (id, school_id, name, description, permissions, is_super_admin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, r.ID, r.SchoolID, r.Name, r.Description, r.Permissions, r.IsSuperAdmin, r.CreatedAt, r.UpdatedAt)
	return err
}