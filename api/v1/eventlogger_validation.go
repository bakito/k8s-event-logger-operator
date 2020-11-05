package v1

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	english "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/translations/en"
	"reflect"
	"regexp"
	"strings"
)

var (
	keyPattern        = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$`)
	labelValuePattern = regexp.MustCompile(`^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$`)
)

// Custom type for context key, so we don't have to use 'string' directly
type contextKey string

var specKey = contextKey("spec")
var errorsKey = contextKey("errors")

// Hash the event
func (in *EventLoggerSpec) Hash() string {
	h := sha256.New()
	bytes, _ := json.Marshal(in)
	_, _ = h.Write(bytes)
	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

// Validate the event
func (in *EventLoggerSpec) Validate() error {
	return newEventLoggerValidator(in).Validate()
}

func k8sLabelValues(_ context.Context, fl validator.FieldLevel) bool {
	if labels, ok := fl.Field().Interface().(map[string]string); ok {
		for _, v := range labels {
			if !labelValuePattern.MatchString(v) {
				return false
			}
		}
	}
	return true
}

func k8sKeys(_ context.Context, fl validator.FieldLevel) bool {
	if annotations, ok := fl.Field().Interface().(map[string]string); ok {
		for k := range annotations {
			if !keyPattern.MatchString(k) {
				return false
			}
		}
	}
	return true
}

// eventLoggerValidator is a custom validator for the event logger
type eventLoggerValidator struct {
	val   *validator.Validate
	ctx   context.Context
	spec  *EventLoggerSpec
	trans ut.Translator
}

// Custom error for event logger validation
type eventLoggerValidatorError struct {
	errList []string
}

func (err eventLoggerValidatorError) Error() string {
	return strings.Join(err.errList, "\n")
}

func (err *eventLoggerValidatorError) addError(errStr string) {
	err.errList = append(err.errList, errStr)
}

// newEventLoggerValidator creates a new EventLoggerValidator
func newEventLoggerValidator(spec *EventLoggerSpec) *eventLoggerValidator {
	result := validator.New()

	_ = result.RegisterValidationCtx("k8s-label-keys", k8sKeys)
	_ = result.RegisterValidationCtx("k8s-label-values", k8sLabelValues)
	_ = result.RegisterValidationCtx("k8s-annotation-keys", k8sKeys)

	// context
	ctx := context.WithValue(context.Background(), specKey, spec)
	ctx = context.WithValue(ctx, errorsKey, &eventLoggerValidatorError{})

	// default translations
	eng := english.New()
	uni := ut.New(eng, eng)
	trans, _ := uni.GetTranslator("en")
	_ = en.RegisterDefaultTranslations(result, trans)

	// additional translations
	translations := []struct {
		tag         string
		translation string
	}{
		{
			tag:         "k8s-label-keys",
			translation: fmt.Sprintf("'key in {0}' must match the pattern %s", keyPattern.String()),
		},
		{
			tag:         "k8s-label-values",
			translation: fmt.Sprintf("'values in {0}' must match the pattern %s", labelValuePattern.String()),
		},
		{
			tag:         "k8s-annotation-keys",
			translation: fmt.Sprintf("'key in {0}' must match the pattern %s", keyPattern.String()),
		},
	}
	for _, t := range translations {
		_ = result.RegisterTranslation(t.tag, trans, registrationFunc(t.tag, t.translation), translateFunc)
	}

	return &eventLoggerValidator{
		val:   result,
		ctx:   ctx,
		spec:  spec,
		trans: trans,
	}
}

func registrationFunc(tag string, translation string) validator.RegisterTranslationsFunc {
	return func(ut ut.Translator) (err error) {
		if err = ut.Add(tag, translation, true); err != nil {
			return
		}
		return
	}
}

func translateFunc(ut ut.Translator, fe validator.FieldError) string {
	t, err := ut.T(fe.Tag(), reflect.ValueOf(fe.Value()).String(), fe.Param())
	if err != nil {
		return fe.(error).Error()
	}
	return t
}

// Validate validates the entire event logger spec for errors and returns an error (it can be casted to
// eventLoggerValidatorError, containing a list of errors inside). When error is printed as string, it will
// automatically contains the full list of validation errors.
func (v *eventLoggerValidator) Validate() error {
	// validate spec
	err := v.val.StructCtx(v.ctx, v.spec)
	if err == nil {
		return nil
	}

	// collect human-readable errors
	result := eventLoggerValidatorError{}
	vErrors := err.(validator.ValidationErrors) // nolint: errcheck
	for _, vErr := range vErrors {
		errStr := fmt.Sprintf("%s: %s", vErr.Namespace(), vErr.Translate(v.trans))
		result.addError(errStr)
	}

	// collect additional errors stored in context
	for _, errStr := range v.ctx.Value(errorsKey).(*eventLoggerValidatorError).errList { // nolint: errcheck
		result.addError(errStr)
	}

	return result
}
