package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

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

type PermissionsValidationOptions struct {
	AllowWildcard bool
	AllowEmpty    bool
}

func (p Permissions) ValidateStrict(opts PermissionsValidationOptions) error {
	if len(p) == 0 {
		if opts.AllowEmpty {
			return nil
		}
		return fmt.Errorf("permissions cannot be empty")
	}

	allowed := GetSuperAdminPermissions()
	for resource, actions := range p {
		if resource == "" {
			return fmt.Errorf("permission resource cannot be empty")
		}
		allowedActions, ok := allowed[resource]
		if !ok {
			return fmt.Errorf("unknown permission resource: %q", resource)
		}

		allowedSet := make(map[string]struct{}, len(allowedActions))
		for _, a := range allowedActions {
			allowedSet[a] = struct{}{}
		}

		if len(actions) == 0 {
			return fmt.Errorf("resource %q must have at least one action", resource)
		}

		for _, action := range actions {
			if action == "" {
				return fmt.Errorf("action for resource %q cannot be empty", resource)
			}
			if action == "*" && !opts.AllowWildcard {
				return fmt.Errorf("wildcard action '*' is not allowed")
			}
			if action == "*" {
				continue
			}
			if _, ok := allowedSet[action]; !ok {
				return fmt.Errorf("invalid action %q for resource %q", action, resource)
			}
		}
	}
	return nil
}

// Role represents a role in the system
type Role struct {
	BaseModel
	SchoolID     uuid.UUID   `json:"school_id" db:"school_id"`
	Name         string      `json:"name" db:"name"`
	Description  string      `json:"description" db:"description"`
	Permissions  Permissions `json:"permissions" db:"permissions"`
	IsSuperAdmin bool        `json:"is_super_admin" db:"is_super_admin"`
}

type RoleResponse struct {
	ID           uuid.UUID   `json:"id"`
	SchoolID     uuid.UUID   `json:"school_id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Permissions  Permissions `json:"permissions"`
	IsSuperAdmin bool        `json:"is_super_admin"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (r *Role) ToResponse() RoleResponse {
	return RoleResponse{
		ID:           r.ID,
		SchoolID:     r.SchoolID,
		Name:         r.Name,
		Description:  r.Description,
		Permissions:  r.Permissions,
		IsSuperAdmin: r.IsSuperAdmin,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
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

// Scan implements the sql.Scanner interface for database scanning
func (p *Permissions) Scan(value interface{}) error {
	if value == nil {
		*p = make(Permissions)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into Permissions", value)
	}

	return json.Unmarshal(bytes, p)
}

// Value implements the driver.Valuer interface for database storage
func (p Permissions) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
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

func ListRoles(dbx DBTX, schoolID uuid.UUID) ([]Role, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	roles := []Role{}
	err := dbx.SelectContext(ctx, &roles, `
		SELECT id, school_id, name, description, permissions, is_super_admin, created_at, updated_at
		FROM roles WHERE school_id = $1 AND deleted_at IS NULL
	`, schoolID)
	return roles, err
}

func GetRole(dbx DBTX, schoolID uuid.UUID, roleID uuid.UUID) (*Role, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	role := Role{}
	err := dbx.GetContext(ctx, &role, `
		SELECT id, school_id, name, description, permissions, is_super_admin, created_at, updated_at
		FROM roles WHERE school_id = $1 AND id = $2 AND deleted_at IS NULL
	`, schoolID, roleID)
	return &role, err
}

func (r *Role) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	r.UpdatedAt = time.Now().UTC()
	_, err := dbx.ExecContext(ctx, `
		UPDATE roles
		SET name = $1,
			description = $2,
			permissions = $3,
			is_super_admin = $4,
			updated_at = $5
		WHERE school_id = $6 AND id = $7
	`, r.Name, r.Description, r.Permissions, r.IsSuperAdmin, r.UpdatedAt, r.SchoolID, r.ID)
	return err
}

func DeleteRole(dbx DBTX, schoolID uuid.UUID, roleID uuid.UUID) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		UPDATE roles
		SET deleted_at = NOW()
		WHERE school_id = $1 AND id = $2 AND deleted_at IS NULL
	`, schoolID, roleID)
	return err
}
