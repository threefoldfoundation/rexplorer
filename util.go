package main

import (
	"fmt"
	"os"
	"strings"
)

func askYesNoQuestion(str string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [Y/N] ", str)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false, fmt.Errorf("failed to scan Y/N response: %v", err)
	}
	response = strings.ToLower(response)
	if containsString(okayResponses, response) {
		return true, nil
	}
	if containsString(nokayResponses, response) {
		return false, nil
	}

	fmt.Fprintln(os.Stderr, "please answer using 'yes' or 'no'")
	return askYesNoQuestion(str)
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

var (
	okayResponses  = []string{"y", "ye", "yes", "ok", "okay"}
	nokayResponses = []string{"n", "no", "noo", "nope"}
)
