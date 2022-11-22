package v1

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/bakito/k8s-event-logger-operator/version"
	english "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/translations/en"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Custom type for context key, so we don't have to use 'string' directly
type contextKey string

var (
	specKey   = contextKey("spec")
	errorsKey = contextKey("errors")
)

// HasChanged check if the spec or operator version has changed
func (in *EventLogger) HasChanged() bool {
	return in.Status.Hash != in.Spec.Hash() || in.Status.OperatorVersion != version.Version
}

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
			if errs := validation.IsValidLabelValue(v); len(errs) > 0 {
				return false
			}
		}
	}
	return true
}

func k8sLabelAnnotationKeys(_ context.Context, fl validator.FieldLevel) bool {
	if annotations, ok := fl.Field().Interface().(map[string]string); ok {
		for k := range annotations {
			if errs := validation.IsQualifiedName(k); len(errs) > 0 {
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

	_ = result.RegisterValidationCtx("k8s-label-annotation-keys", k8sLabelAnnotationKeys)
	_ = result.RegisterValidationCtx("k8s-label-values", k8sLabelValues)

	errKey := strings.Join(validation.IsQualifiedName("a@a"), " ")
	errLabelVal := strings.Join(validation.IsValidLabelValue("a:/a"), " ")

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
			tag:         "k8s-label-annotation-keys",
			translation: fmt.Sprintf("'key in {0}' must match the pattern %s", errKey),
		},
		{
			tag:         "k8s-label-values",
			translation: fmt.Sprintf("'values in {0}' must match the pattern %s", errLabelVal),
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
// automatically contain the full list of validation errors.
func (v *eventLoggerValidator) Validate() error {
	// validate spec
	err := v.val.StructCtx(v.ctx, v.spec)
	if err == nil {
		return nil
	}

	// collect human-readable errors
	result := eventLoggerValidatorError{}
	var vErrors validator.ValidationErrors
	errors.As(err, &vErrors)
	for _, vErr := range vErrors {
		errStr := fmt.Sprintf("%s: %s", vErr.Namespace(), vErr.Translate(v.trans))
		result.addError(errStr)
	}

	// collect additional errors stored in context
	for _, errStr := range v.ctx.Value(errorsKey).(*eventLoggerValidatorError).errList {
		result.addError(errStr)
	}

	return result
}
