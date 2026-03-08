package middleware

import (
	"github.com/go-playground/validator/v10"
	"github.com/nyaruka/phonenumbers"
)

// PhoneValidator checks if the string is a valid E.164 phone number
var PhoneValidator validator.Func = func(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	
	// We pass "ZZ" as the region to force the user to provide a country code (e.g., +234...)
	num, err := phonenumbers.Parse(phone, "ZZ")
	if err != nil {
		return false
	}
	
	return phonenumbers.IsValidNumber(num)
}