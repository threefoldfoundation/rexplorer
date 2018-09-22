package main

import "testing"

func TestContainsString(t *testing.T) {
	contains := containsString(okayResponses, "foo")
	if contains {
		t.Error("okayResponses doesn't contains foo, but containsString returns true")
	}
	contains = containsString(nokayResponses, "foo")
	if contains {
		t.Error("nokayResponses doesn't contains foo, but containsString returns true")
	}
	for _, resp := range okayResponses {
		contains = containsString(okayResponses, resp)
		if !contains {
			t.Errorf("okayResponses contains %s but containsString returns false", resp)
		}
	}
	for _, resp := range nokayResponses {
		contains = containsString(nokayResponses, resp)
		if !contains {
			t.Errorf("nokayResponses contains %s but containsString returns false", resp)
		}
	}
}
