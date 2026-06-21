package validators

import (
	"errors"
	"strings"
	"testing"

	"github.com/erick/curriculo/internal/models"
)

func TestValidateMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		max       int
		wantErr   bool
		wantField string
	}{
		{
			name:    "within limit",
			field:   "name",
			value:   "John",
			max:     200,
			wantErr: false,
		},
		{
			name:    "exactly at limit",
			field:   "name",
			value:   strings.Repeat("a", 200),
			max:     200,
			wantErr: false,
		},
		{
			name:      "exceeds limit",
			field:     "name",
			value:     strings.Repeat("a", 201),
			max:       200,
			wantErr:   true,
			wantField: "name",
		},
		{
			name:    "empty string",
			field:   "name",
			value:   "",
			max:     200,
			wantErr: false,
		},
		{
			name:      "unicode chars count correctly",
			field:     "name",
			value:     strings.Repeat("é", 201),
			max:       200,
			wantErr:   true,
			wantField: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMaxLength(tt.field, tt.value, tt.max)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var ve *models.ValidationError
				if !errors.As(err, &ve) {
					t.Fatal("expected ValidationError type")
				}
				if _, ok := ve.Fields[tt.wantField]; !ok {
					t.Errorf("expected field %q in validation error", tt.wantField)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateMinLength(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		min       int
		wantErr   bool
		wantField string
	}{
		{
			name:    "above minimum",
			field:   "password",
			value:   "12345678",
			min:     8,
			wantErr: false,
		},
		{
			name:    "exactly at minimum",
			field:   "password",
			value:   "12345678",
			min:     8,
			wantErr: false,
		},
		{
			name:      "below minimum",
			field:     "password",
			value:     "1234567",
			min:       8,
			wantErr:   true,
			wantField: "password",
		},
		{
			name:      "empty string",
			field:     "password",
			value:     "",
			min:       1,
			wantErr:   true,
			wantField: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMinLength(tt.field, tt.value, tt.min)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var ve *models.ValidationError
				if !errors.As(err, &ve) {
					t.Fatal("expected ValidationError type")
				}
				if _, ok := ve.Fields[tt.wantField]; !ok {
					t.Errorf("expected field %q in validation error", tt.wantField)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{name: "valid email", email: "user@example.com", wantErr: false},
		{name: "valid with subdomain", email: "user@mail.example.com", wantErr: false},
		{name: "valid with plus", email: "user+tag@example.com", wantErr: false},
		{name: "valid with dots", email: "first.last@example.com", wantErr: false},
		{name: "missing at sign", email: "userexample.com", wantErr: true},
		{name: "missing domain", email: "user@", wantErr: true},
		{name: "missing local part", email: "@example.com", wantErr: true},
		{name: "empty string", email: "", wantErr: true},
		{name: "spaces in email", email: "user @example.com", wantErr: true},
		{name: "missing tld", email: "user@example", wantErr: true},
		{name: "single char tld", email: "user@example.c", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var ve *models.ValidationError
				if !errors.As(err, &ve) {
					t.Fatal("expected ValidationError type")
				}
				if _, ok := ve.Fields["email"]; !ok {
					t.Error("expected 'email' field in validation error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{name: "non-empty value", field: "name", value: "John", wantErr: false},
		{name: "empty string", field: "name", value: "", wantErr: true},
		{name: "whitespace only", field: "name", value: "   ", wantErr: true},
		{name: "tabs only", field: "name", value: "\t\t", wantErr: true},
		{name: "newlines only", field: "name", value: "\n\n", wantErr: true},
		{name: "value with spaces around", field: "name", value: " hello ", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.field, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var ve *models.ValidationError
				if !errors.As(err, &ve) {
					t.Fatal("expected ValidationError type")
				}
				if _, ok := ve.Fields[tt.field]; !ok {
					t.Errorf("expected field %q in validation error", tt.field)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateTemplateChoice(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{name: "moderno valid", template: "moderno", wantErr: false},
		{name: "classico valid", template: "classico", wantErr: false},
		{name: "minimalista valid", template: "minimalista", wantErr: false},
		{name: "invalid template", template: "fancy", wantErr: true},
		{name: "empty string", template: "", wantErr: true},
		{name: "case sensitive", template: "Moderno", wantErr: true},
		{name: "with spaces", template: " moderno ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplateChoice(tt.template)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var ve *models.ValidationError
				if !errors.As(err, &ve) {
					t.Fatal("expected ValidationError type")
				}
				if _, ok := ve.Fields["template_name"]; !ok {
					t.Error("expected 'template_name' field in validation error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
