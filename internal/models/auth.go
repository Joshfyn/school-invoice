package models

// Register is the request body for school registration
type Register struct {
	// School details
	SchoolName    string `json:"school_name" binding:"required,min=2,max=255"`
	Subdomain     string `json:"subdomain" binding:"required,min=3,max=50,alphanum"`
	SchoolPhone   string `json:"school_phone" binding:"required"`
	SchoolEmail   string `json:"school_email" binding:"required,email"`
	SchoolAddress string `json:"school_address" binding:"required"`

	// Super Admin details
	AdminEmail     string `json:"admin_email" binding:"required,email"`
	AdminPassword  string `json:"admin_password" binding:"required,min=8"`
	AdminFirstName string `json:"admin_first_name" binding:"required,min=2"`
	AdminLastName  string `json:"admin_last_name" binding:"required,min=2"`
	AdminPhone     string `json:"admin_phone" binding:"required"`
}

func (r *Register) DomainExists(dbx DBTX) (bool, error) {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	exists := false
	err := dbx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schools WHERE subdomain = $1)", r.Subdomain).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
