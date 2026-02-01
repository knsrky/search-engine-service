// Package validator provides request validation using go-playground/validator.
package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the go-playground validator with custom configuration.
type Validator struct {
	v *validator.Validate
}

// ValidationError represents a single field validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

// Error implements the error interface.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, e := range ve {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(e.Message)
	}
	return sb.String()
}

// New creates a new Validator instance with custom tag name and validations.
func New() *Validator {
	v := validator.New()

	// Use JSON tag names for field names in errors
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return fld.Name
		}
		return name
	})

	return &Validator{v: v}
}

// Validate validates the given struct and returns ValidationErrors if invalid.
func (v *Validator) Validate(i interface{}) error {
	err := v.v.Struct(i)
	if err == nil {
		return nil
	}

	// Convert to ValidationErrors
	var errs ValidationErrors
	for _, e := range err.(validator.ValidationErrors) {
		errs = append(errs, ValidationError{
			Field:   e.Field(),
			Tag:     e.Tag(),
			Value:   fmt.Sprintf("%v", e.Value()),
			Message: formatErrorMessage(e),
		})
	}

	return errs
}

// formatErrorMessage generates a human-readable error message.
func formatErrorMessage(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	default:
		return fmt.Sprintf("%s failed %s validation", field, e.Tag())
	}
}
