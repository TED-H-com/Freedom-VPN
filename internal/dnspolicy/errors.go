// Package dnspolicy implements the Freedom VPN domain policy engine, list
// source parsers, deterministic bundle compiler, and AdGuard Home backend
// adapter (§12, DNS01–DNS04, SEC02, AT20–AT22).
package dnspolicy

import (
	"errors"
	"fmt"
)

// ValidationErrorCode identifies a machine-readable reason code suitable for
// returning to the Admin/Member UI and API error details (§12.4, §14.6).
type ValidationErrorCode string

const (
	// Domain normalization error codes (§12.4).
	ErrCodeDomainEmpty          ValidationErrorCode = "DOMAIN_EMPTY"
	ErrCodeDomainControlChar    ValidationErrorCode = "DOMAIN_CONTROL_CHAR"
	ErrCodeDomainHasScheme      ValidationErrorCode = "DOMAIN_HAS_SCHEME"
	ErrCodeDomainHasUserInfo    ValidationErrorCode = "DOMAIN_HAS_USERINFO"
	ErrCodeDomainHasPort        ValidationErrorCode = "DOMAIN_HAS_PORT"
	ErrCodeDomainHasPath        ValidationErrorCode = "DOMAIN_HAS_PATH"
	ErrCodeDomainHasQuery       ValidationErrorCode = "DOMAIN_HAS_QUERY"
	ErrCodeDomainHasFragment    ValidationErrorCode = "DOMAIN_HAS_FRAGMENT"
	ErrCodeDomainHasWildcard    ValidationErrorCode = "DOMAIN_HAS_WILDCARD"
	ErrCodeDomainEmptyLabel     ValidationErrorCode = "DOMAIN_EMPTY_LABEL"
	ErrCodeDomainLabelTooLong   ValidationErrorCode = "DOMAIN_LABEL_TOO_LONG"
	ErrCodeDomainTooLong        ValidationErrorCode = "DOMAIN_TOO_LONG"
	ErrCodeDomainInvalidIDNA    ValidationErrorCode = "DOMAIN_INVALID_IDNA"
	ErrCodeDomainSingleLabel    ValidationErrorCode = "DOMAIN_SINGLE_LABEL"
	ErrCodeDomainNumericTLD     ValidationErrorCode = "DOMAIN_NUMERIC_TLD"

	// Rule and draft validation error codes (§12.4, §12.5).
	ErrCodeProfileIDEmpty          ValidationErrorCode = "PROFILE_ID_EMPTY"
	ErrCodeInvalidRevision         ValidationErrorCode = "INVALID_REVISION"
	ErrCodeRuleIDEmpty             ValidationErrorCode = "RULE_ID_EMPTY"
	ErrCodeRuleIDDuplicate         ValidationErrorCode = "RULE_ID_DUPLICATE"
	ErrCodeProfileRefMismatch      ValidationErrorCode = "PROFILE_REF_MISMATCH"
	ErrCodeRuleInvalidAction       ValidationErrorCode = "RULE_INVALID_ACTION"
	ErrCodeRuleInvalidLayer        ValidationErrorCode = "RULE_INVALID_LAYER"
	ErrCodeSameLayerConflict       ValidationErrorCode = "SAME_LAYER_CONFLICT"
	ErrCodeSourceNotFound          ValidationErrorCode = "SOURCE_NOT_FOUND"
	ErrCodeSourceLicenseUnapproved ValidationErrorCode = "SOURCE_LICENSE_UNAPPROVED"
	ErrCodeSourceInvalidURL        ValidationErrorCode = "SOURCE_INVALID_URL"
	ErrCodeSourceTooLarge          ValidationErrorCode = "SOURCE_TOO_LARGE"
	ErrCodeSourceEmpty             ValidationErrorCode = "SOURCE_EMPTY"
	ErrCodeSourceFormatUnsupported ValidationErrorCode = "SOURCE_FORMAT_UNSUPPORTED"
	ErrCodeRuleLimitExceeded       ValidationErrorCode = "RULE_LIMIT_EXCEEDED"

	// Bundle compilation and verification error codes (§12.5, §12.7).
	ErrCodeBundleInvalidWindow  ValidationErrorCode = "BUNDLE_INVALID_VALIDITY_WINDOW"
	ErrCodeBundleExpired        ValidationErrorCode = "BUNDLE_EXPIRED"
	ErrCodeBundleHashMismatch   ValidationErrorCode = "BUNDLE_HASH_MISMATCH"
	ErrCodeBundlePrivacyInvalid ValidationErrorCode = "BUNDLE_PRIVACY_INVALID"
	ErrCodeBundleNoUpstreams    ValidationErrorCode = "BUNDLE_NO_UPSTREAMS"
	ErrCodeBundleSyntaxInvalid  ValidationErrorCode = "BUNDLE_RULE_SYNTAX_INVALID"
)

// ValidationError is a typed validation error carrying a UI-consumable reason
// code, field name, offending input, and human-readable message.
type ValidationError struct {
	Code    ValidationErrorCode `json:"code"`
	Field   string              `json:"field,omitempty"`
	Input   string              `json:"input,omitempty"`
	Message string              `json:"message"`
	Cause   error               `json:"-"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field != "" {
		return fmt.Sprintf("%s (%s): %s", e.Code, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error cause, if any.
func (e *ValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// ErrorCode extracts the ValidationErrorCode from err if it is or wraps a
// *ValidationError, or returns "" otherwise.
func ErrorCode(err error) ValidationErrorCode {
	var ve *ValidationError
	if errors.As(err, &ve) && ve != nil {
		return ve.Code
	}
	return ""
}

func newDomainError(code ValidationErrorCode, input, msg string, cause error) error {
	return &ValidationError{
		Code:    code,
		Field:   "domain",
		Input:   input,
		Message: msg,
		Cause:   cause,
	}
}
