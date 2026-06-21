package validators

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/erick/curriculo/internal/models"
)

// Field length limits as defined in requirements.
const (
	MaxNameLength           = 200
	MaxTitleLength          = 200
	MaxSummaryLength        = 2000
	MaxExperienceDescLength = 5000
	MaxSkillNameLength      = 100
)

// validTemplates defines the allowed resume template choices.
var validTemplates = map[string]bool{
	"moderno":     true,
	"classico":    true,
	"minimalista": true,
}

// emailRegex is a basic email format validation pattern.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateMaxLength returns a ValidationError if value exceeds max characters.
func ValidateMaxLength(field, value string, max int) error {
	if len([]rune(value)) > max {
		return &models.ValidationError{
			Fields: map[string]string{
				field: fmt.Sprintf("%s must be at most %d characters", field, max),
			},
		}
	}
	return nil
}

// ValidateMinLength returns a ValidationError if value is shorter than min characters.
func ValidateMinLength(field, value string, min int) error {
	if len([]rune(value)) < min {
		return &models.ValidationError{
			Fields: map[string]string{
				field: fmt.Sprintf("%s must be at least %d characters", field, min),
			},
		}
	}
	return nil
}

// ValidateEmail performs basic email format validation.
func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return &models.ValidationError{
			Fields: map[string]string{
				"email": "invalid email format",
			},
		}
	}
	return nil
}

// ValidateRequired checks that the value is non-empty after trimming whitespace.
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &models.ValidationError{
			Fields: map[string]string{
				field: fmt.Sprintf("%s is required", field),
			},
		}
	}
	return nil
}

// ValidateTemplateChoice validates that template is one of the allowed choices:
// moderno, classico, minimalista.
func ValidateTemplateChoice(template string) error {
	if !validTemplates[template] {
		return &models.ValidationError{
			Fields: map[string]string{
				"template_name": "template must be one of: moderno, classico, minimalista",
			},
		}
	}
	return nil
}
