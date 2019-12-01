package event

import (
	"testing"

	. "gotest.tools/assert"
)

func Test_matches(t *testing.T) {
	lp := &loggingPredicate{}

	Assert(t, lp.matches([]string{"abc", "xyz"}, "abc"))
	Assert(t, lp.matches([]string{"abc", "xyz"}, "^abc$"))
	Assert(t, !lp.matches([]string{"abc", "xyz"}, "^ab$"))
}

func Test_contains(t *testing.T) {
	lp := &loggingPredicate{}

	Assert(t, lp.contains([]string{"abc", "xyz"}, "abc"))
	Assert(t, lp.contains([]string{"abc", "xyz"}, "xyz"))
	Assert(t, !lp.contains([]string{"abc", "xyz"}, "xxx"))
}
