package types

import (
	"sort"
	"testing"
)

func TestDescriptionFilterMatch(t *testing.T) {
	testMatch := func(pattern string, strs ...string) {
		t.Helper()
		filter := mustNewDescriptionFilter(t, pattern)
		for _, str := range strs {
			if !filter.Match(str) {
				t.Errorf("failed to match pattern %q on string %q", pattern, str)
			}
		}
	}
	testNotMatch := func(pattern string, strs ...string) {
		t.Helper()
		filter := mustNewDescriptionFilter(t, pattern)
		for _, str := range strs {
			if filter.Match(str) {
				t.Errorf("managed to match pattern %q on string %q unexpectantly", pattern, str)
			}
		}
	}

	testMatch("", "")
	testNotMatch("", "a")

	testMatch("foo", "foo")
	testNotMatch("foo", "dfoo", "doo", "oof", "food")

	testMatch("?at", "cat", "dat", "fat")
	testNotMatch("?at", "at", "acat")

	testMatch("[abc]at", "aat", "bat", "cat")
	testNotMatch("[abc]at", "fat", "at")

	testMatch("[!abc]at", "dat", "vat", "zat")
	testNotMatch("[!abc]at", "bat", "at", "cat", "aat", "adat")

	testMatch("[!abc]at", "dat", "vat", "zat")
	testNotMatch("[!abc]at", "bat", "at", "cat", "aat", "adat")

	testMatch("{cat,bat,[fr]at,[v-z]at}", "cat", "bat", "fat", "rat", "vat", "zat", "xat")
	testNotMatch("{cat,bat,[fr]at,[v-z]at}", "at", "iat", "acat")

	testMatch("reward:{block,tx}", "reward:block", "reward:tx")
}

func TestDescriptionFilterSetMatch(t *testing.T) {
	set, err := NewDescriptionFilterSet("cat", "bat", "[fr]at", "[v-z]at")
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}

	for _, str := range []string{"cat", "bat", "fat", "rat", "vat", "zat", "xat"} {
		if !set.Match(str) {
			t.Errorf("failed to match string %q", str)
		}
	}
	for _, str := range []string{"at", "iat", "acat"} {
		if set.Match(str) {
			t.Errorf("managed to match string %q unexpectantly", str)
		}
	}

	set, err = NewDescriptionFilterSet("foo:*", "bar:*")
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}

	for _, str := range []string{"foo:", "bar:", "foo: hallo", "bar: world"} {
		if !set.Match(str) {
			t.Errorf("failed to match string %q", str)
		}
	}
	for _, str := range []string{"foo bar", "bar foo", "foo", "bar"} {
		if set.Match(str) {
			t.Errorf("managed to match string %q unexpectantly", str)
		}
	}
}

func mustNewDescriptionFilter(t *testing.T, pattern string) *DescriptionFilter {
	t.Helper()
	filter, err := NewDescriptionFilter(pattern)
	if err != nil {
		t.Fatal("failed to create description filter: ", err)
	}
	return filter
}

func TestDescriptionFilterSetSorting(t *testing.T) {
	set, err := NewDescriptionFilterSet("c", "e", "d", "b", "a")
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}

	if n := set.Len(); n != 5 {
		t.Fatal("unexpected length: ", n)
	}
	sort.Sort(set)
	if n := set.Len(); n != 5 {
		t.Fatal("unexpected length: ", n)
	}

	expectedOrder := []string{"a", "b", "c", "d", "e"}
	for i, expectedPattern := range expectedOrder {
		if set.filters[i].pattern != expectedPattern {
			t.Errorf("unexpected pattern on index %d: %s", i, set.filters[i].pattern)
		}
	}
}

func TestDescriptionFilterSetDifference(t *testing.T) {
	setA, err := NewDescriptionFilterSet("1", "2", "3")
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}
	setB, err := NewDescriptionFilterSet("2", "3", "4")
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}

	// setA \ setB
	setC := setA.Difference(setB)
	expectedSet := mustNewDescriptionFilterSet(t, "1")
	if !expectedSet.Equals(setC) {
		t.Fatal(expectedSet.String(), "!=", setC.String())
	}
	// setB \ setA
	setC = setB.Difference(setA)
	expectedSet = mustNewDescriptionFilterSet(t, "4")
	if !expectedSet.Equals(setC) {
		t.Fatal(expectedSet.String(), "!=", setC.String())
	}
}

func TestEmptyDescriptionFilterSetDifference(t *testing.T) {
	emptyA := mustNewDescriptionFilterSet(t)
	emptyB := mustNewDescriptionFilterSet(t)
	setA := mustNewDescriptionFilterSet(t, "4", "2")

	// empty \ empty
	emptyC := emptyA.Difference(emptyB)
	expectedSet := mustNewDescriptionFilterSet(t)
	if !emptyC.Equals(expectedSet) {
		t.Fatal(expectedSet.String(), "!=", emptyC.String())
	}
	emptyC = emptyB.Difference(emptyA)
	expectedSet = mustNewDescriptionFilterSet(t)
	if !emptyC.Equals(expectedSet) {
		t.Fatal(expectedSet.String(), "!=", emptyC.String())
	}

	// empty \ set
	setB := emptyA.Difference(setA)
	expectedSet = mustNewDescriptionFilterSet(t)
	if !setB.Equals(expectedSet) {
		t.Fatal(expectedSet.String(), "!=", setB.String())
	}

	// set \ empty
	setB = setA.Difference(emptyA)
	expectedSet = mustNewDescriptionFilterSet(t, "2", "4")
	if !setB.Equals(expectedSet) {
		t.Fatal(expectedSet.String(), "!=", setB.String())
	}
}

func mustNewDescriptionFilterSet(t *testing.T, patterns ...string) DescriptionFilterSet {
	t.Helper()
	set, err := NewDescriptionFilterSet(patterns...)
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}
	return set
}

func TestDescriptionFilterSetString(t *testing.T) {
	set, err := NewDescriptionFilterSet("c", "e f", "d", "b*a 1", "a")
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}
	const expectedStr = "c,e f,d,b*a 1,a"
	str := set.String()
	if expectedStr != str {
		t.Fatal("unexpected stringified DescriptionFilterSet: ", str)
	}
}

func TestEmptyDescriptionFilterSetString(t *testing.T) {
	set, err := NewDescriptionFilterSet()
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}
	str := set.String()
	if len(str) != 0 {
		t.Fatal("unexpected stringified DescriptionFilterSet: ", str)
	}
}

func TestLoadStringDescriptionFilterSet(t *testing.T) {
	testCases := []struct {
		String                     string
		ExpectedMatchingStrings    []string
		ExpectedNonMatchingStrings []string
	}{
		{
			"",
			nil,
			nil,
		},
		{
			"foo:*,bar:*",
			[]string{"foo:", "bar:", "foo: hallo", "bar: world"},
			[]string{"foo bar", "bar foo", "foo", "bar"},
		},
		{
			`foo:*,"bar, *"`,
			[]string{"foo:", "bar, ", "foo: hallo", "bar, world"},
			[]string{"foo bar", "bar,foo", "foo", "bar,"},
		},
	}
	for caseIndex, testCase := range testCases {
		var set DescriptionFilterSet
		// try to load the string
		err := set.LoadString(testCase.String)
		if err != nil {
			t.Errorf("%d) failed to load string %s: %v", caseIndex, testCase.String, err)
			continue
		}

		// test matching and non-matching strings
		for _, str := range testCase.ExpectedMatchingStrings {
			if !set.Match(str) {
				t.Errorf("%d) failed to match string %q with pattern %q", caseIndex, str, testCase.String)
			}
		}
		for _, str := range testCase.ExpectedNonMatchingStrings {
			if set.Match(str) {
				t.Errorf("%d) managed to match string %q unexpectantly with pattern %q", caseIndex, str, testCase.String)
			}
		}
		// stringify again and make sure it is the same string
		stringifiedTestCase := set.String()
		if stringifiedTestCase != testCase.String {
			t.Errorf("%d) unexpected stringified test case: %s != %s", caseIndex, stringifiedTestCase, testCase.String)
		}
	}
}

func TestDescriptionFilterSetComplexity(t *testing.T) {
	set, err := NewDescriptionFilterSet(`"foo":*`, `foo,*`)
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}

	var otherSet DescriptionFilterSet
	err = otherSet.LoadString(set.String())
	if err != nil {
		t.Fatal("failed to load stringified filter set as a new set: ", err)
	}
	if !set.Equals(otherSet) {
		t.Fatal(set.String(), "!=", otherSet.String())
	}

	for _, str := range []string{"foo,", `"foo":`, "foo, hallo", `"foo": hallo`} {
		if !set.Match(str) {
			t.Errorf("failed to match string %q", str)
		}
	}
	for _, str := range []string{"foo bar", `"foo`, `"foo:`, `"foo" hallo`, "foo", "bar", "foo: bar"} {
		if set.Match(str) {
			t.Errorf("managed to match string %q unexpectantly", str)
		}
	}
}

func TestDescriptionFilterSetAppendPattern(t *testing.T) {
	set, err := NewDescriptionFilterSet("a", "b", "c")
	if err != nil {
		t.Fatal("failed to create description filter set: ", err)
	}

	// valid and unique pattern
	if err = set.AppendPattern("d"); err != nil {
		t.Fatal("failed to append pattern 'd', even though it is valid and unique:", err)
	}

	// valid but duplicate pattern
	if err = set.AppendPattern("a"); err == nil {
		t.Fatal("managed to append pattern 'a', even though it isn't unique:", set.String())
	}
	if err = set.AppendPattern("b"); err == nil {
		t.Fatal("managed to append pattern 'b', even though it isn't unique:", set.String())
	}
	if err = set.AppendPattern("c"); err == nil {
		t.Fatal("managed to append pattern 'c', even though it isn't unique:", set.String())
	}

	// invalid pattern, even though it is unique
	if err = set.AppendPattern("[a"); err == nil {
		t.Fatal("managed to append pattern '[a', even though it isn't valid:", set.String())
	}
}

func (set DescriptionFilterSet) Equals(other DescriptionFilterSet) bool {
	if len(set.filters) != len(other.filters) {
		return false
	}
	for i, filter := range set.filters {
		if filter.pattern != other.filters[i].pattern {
			return false
		}
	}
	return true
}
