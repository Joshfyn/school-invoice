package middleware

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/nyaruka/phonenumbers"
)

const defaultPhoneRegion = "NG"

// ParsePhone parses a phone number, accepting E.164 (+234...) or local Nigerian formats (080...).
func ParsePhone(phone string) (*phonenumbers.PhoneNumber, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil, phonenumbers.ErrInvalidCountryCode
	}

	num, err := phonenumbers.Parse(phone, "ZZ")
	if err != nil {
		num, err = phonenumbers.Parse(phone, defaultPhoneRegion)
	}
	if err != nil {
		return nil, err
	}
	if !phonenumbers.IsValidNumber(num) {
		return nil, phonenumbers.ErrInvalidCountryCode
	}
	return num, nil
}

// FormatPhoneE164 normalizes a phone number to E.164 (e.g. +2348012345678).
func FormatPhoneE164(phone string) string {
	num, err := ParsePhone(phone)
	if err != nil {
		return strings.TrimSpace(phone)
	}
	return phonenumbers.Format(num, phonenumbers.E164)
}

// PhoneValidator checks if the string is a valid phone number (E.164 or Nigerian local).
var PhoneValidator validator.Func = func(fl validator.FieldLevel) bool {
	_, err := ParsePhone(fl.Field().String())
	return err == nil
}
