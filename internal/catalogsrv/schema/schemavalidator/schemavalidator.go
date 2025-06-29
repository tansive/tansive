package schemavalidator

import "github.com/go-playground/validator/v10"

var schemaValidator *validator.Validate

func V() *validator.Validate {
	if schemaValidator == nil {
		schemaValidator = validator.New(validator.WithRequiredStructEnabled())
	}
	return schemaValidator
}
