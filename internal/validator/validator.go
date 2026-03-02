package validator

import (
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct validates a struct and returns a map of field errors.
func ValidateStruct(s interface{}) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
	errors := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		errors[err.Field()] = "failed validation: " + err.Tag()
	}
	return errors
}
