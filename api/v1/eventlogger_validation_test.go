package v1

import (
	is "gotest.tools/assert/cmp"
	"testing"

	. "gotest.tools/assert"
)

func Test_Validate_Success(t *testing.T) {
	s := &EventLoggerSpec{
		Annotations: map[string]string{"valid": "valid"},
		Labels:      map[string]string{"valid": "valid"},
	}

	Assert(t, is.Nil(s.Validate()))
}

func Test_Validate_Invalid_LabelKey(t *testing.T) {
	s := &EventLoggerSpec{
		Labels: map[string]string{"in valid": "valid"},
	}

	Assert(t, s.Validate() != nil)
}

func Test_Validate_Invalid_LabelValue(t *testing.T) {
	s := &EventLoggerSpec{
		Labels: map[string]string{"valid": "in valid"},
	}

	Assert(t, s.Validate() != nil)
}

func Test_Validate_Invalid_AnnotationKey(t *testing.T) {
	s := &EventLoggerSpec{
		Annotations: map[string]string{"in valid": "valid"},
	}

	Assert(t, s.Validate() != nil)
}
