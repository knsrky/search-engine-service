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

	// Register custom validations
	registerCustomValidations(v)

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

// ValidateVar validates a single variable against a tag.
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.v.Var(field, tag)
}

// registerCustomValidations registers custom validation functions.
func registerCustomValidations(v *validator.Validate) {
	// Content type validation
	_ = v.RegisterValidation("content_type", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return value == "" || value == "video" || value == "article"
	})

	// Sort field validation
	_ = v.RegisterValidation("sort_field", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		validFields := []string{"", "relevance", "score", "published_at"}
		for _, f := range validFields {
			if value == f {
				return true
			}
		}
		return false
	})

	// Sort order validation
	_ = v.RegisterValidation("sort_order", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return value == "" || value == "asc" || value == "desc"
	})
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
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "content_type":
		return fmt.Sprintf("%s must be 'video' or 'article'", field)
	case "sort_field":
		return fmt.Sprintf("%s must be one of: relevance, score, published_at", field)
	case "sort_order":
		return fmt.Sprintf("%s must be 'asc' or 'desc'", field)
	default:
		return fmt.Sprintf("%s failed %s validation", field, e.Tag())
	}
}
