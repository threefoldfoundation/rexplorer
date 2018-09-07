package rflag

import (
	"testing"

	"github.com/threefoldfoundation/rexplorer/pkg/types"
)

func TestDescriptionFilterSetFlag(t *testing.T) {
	var set types.DescriptionFilterSet
	flag := descriptionFilterSetFlag{set: &set}

	str := flag.String()
	if len(str) != 0 {
		t.Fatal("unexpected stringified DescriptionFilterSet: ", str)
	}

	flag.Set("foo:*")
	expectedStr := "foo:*"
	str = flag.String()
	if expectedStr != str {
		t.Fatal("stringified DescriptionFilterSet unexpected:", expectedStr, "!=", str)
	}

	flag.Set("bar:*")
	expectedStr = "foo:*,bar:*"
	str = flag.String()
	if expectedStr != str {
		t.Fatal("stringified DescriptionFilterSet unexpected:", expectedStr, "!=", str)
	}

	for _, expectedMatchStr := range []string{"foo: Hello!", "bar:world", "foo:bar", "bar:foo"} {
		if !set.Match(expectedMatchStr) {
			t.Errorf("expected to match %q but failed to do so", expectedMatchStr)
		}
	}
	for _, expectedNotMatchStr := range []string{"", "bar", "foo", "bar foo", "foo bar", "Hello World!"} {
		if set.Match(expectedNotMatchStr) {
			t.Errorf("expected to not match %q but managed to do so", expectedNotMatchStr)
		}
	}
}
