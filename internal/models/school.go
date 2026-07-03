package models

import "time"

// School represents a school (tenant) in the system
type School struct {
	BaseModel
	Name              string  `json:"name" db:"name"`
	Subdomain         string  `json:"subdomain" db:"subdomain"`
	Phone             string  `json:"phone" db:"phone"`
	Email             string  `json:"email" db:"email"`
	Address           string  `json:"address" db:"address"`
	LogoURL           *string `json:"logo_url,omitempty" db:"logo_url"`
	BankName          *string `json:"bank_name,omitempty" db:"bank_name"`
	BankAccountName   *string `json:"bank_account_name,omitempty" db:"bank_account_name"`
	BankAccountNumber *string `json:"bank_account_number,omitempty" db:"bank_account_number"`
	IsActive          bool    `json:"is_active" db:"is_active"`
}

func (s *School) Create(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	_, err := dbx.ExecContext(ctx, `
		INSERT INTO schools (id, name, subdomain, phone, email, address, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, s.ID, s.Name, s.Subdomain, s.Phone, s.Email, s.Address, s.IsActive, s.CreatedAt, s.UpdatedAt)
	return err
}

// Get School Profile
func (s *School) GetProfile(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	err := dbx.QueryRowContext(ctx, `
		SELECT id, name, subdomain, phone, email, address, logo_url,
		       bank_name, bank_account_name, bank_account_number, is_active, created_at, updated_at
		FROM schools WHERE id = $1
	`, s.ID).Scan(&s.ID, &s.Name, &s.Subdomain, &s.Phone, &s.Email, &s.Address, &s.LogoURL,
		&s.BankName, &s.BankAccountName, &s.BankAccountNumber, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	return err
}

// Update School Profile
func (s *School) Update(dbx DBTX) error {
	ctx, cancel := GetDBContext(dbx)
	defer cancel()

	// Build dynamic update query
	query := "UPDATE schools SET updated_at = $1"
	args := []interface{}{time.Now().UTC()}
	argIndex := 2

	if s.Name != "" {
		query += ", name = $" + string(rune('0'+argIndex))
		args = append(args, s.Name)
		argIndex++
	}
	if s.Phone != "" {
		query += ", phone = $" + string(rune('0'+argIndex))
		args = append(args, s.Phone)
		argIndex++
	}
	if s.Email != "" {
		query += ", email = $" + string(rune('0'+argIndex))
		args = append(args, s.Email)
		argIndex++
	}
	if s.Address != "" {
		query += ", address = $" + string(rune('0'+argIndex))
		args = append(args, s.Address)
		argIndex++
	}
	if s.LogoURL != nil {
		query += ", logo_url = $" + string(rune('0'+argIndex))
		args = append(args, s.LogoURL)
		argIndex++
	}
	if s.BankName != nil {
		query += ", bank_name = $" + string(rune('0'+argIndex))
		args = append(args, s.BankName)
		argIndex++
	}
	if s.BankAccountName != nil {
		query += ", bank_account_name = $" + string(rune('0'+argIndex))
		args = append(args, s.BankAccountName)
		argIndex++
	}
	if s.BankAccountNumber != nil {
		query += ", bank_account_number = $" + string(rune('0'+argIndex))
		args = append(args, s.BankAccountNumber)
		argIndex++
	}

	query += " WHERE id = $" + string(rune('0'+argIndex))
	args = append(args, s.ID)

	_, err := dbx.ExecContext(ctx, query, args...)
	return err
}
