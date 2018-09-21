package types

import (
	"bytes"
	"encoding/csv"
	fmt "fmt"
	"io"
	"sort"
	"strings"

	"github.com/gobwas/glob"
)

type (
	// DescriptionFilter is a filter that utilizes glob string pattern matching,
	// such that it will Match some descriptions, while not Match others.
	DescriptionFilter struct {
		glob    glob.Glob
		pattern string
	}

	// DescriptionFilterSet is a set of description filters.
	// It can be used to match a string for any of the description filters part of this set.
	DescriptionFilterSet struct {
		filters []*DescriptionFilter
	}
)

type filterInterface interface {
	Match(string) bool
}

var (
	_ filterInterface = (*DescriptionFilter)(nil)

	_ filterInterface = (*DescriptionFilterSet)(nil)
	_ sort.Interface  = (*DescriptionFilterSet)(nil)
)

// NewDescriptionFilter creates a new description filter using the given glob pattern.
func NewDescriptionFilter(pattern string) (*DescriptionFilter, error) {
	glob, err := glob.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to create description filter: %v", err)
	}
	return &DescriptionFilter{
		glob:    glob,
		pattern: pattern,
	}, nil
}

// Match returns true if the given string matches the DescriptionFilter.
func (df *DescriptionFilter) Match(str string) bool {
	return df.glob.Match(str)
}

// String returns the pattern used for this DescriptionFilter.
func (df *DescriptionFilter) String() string {
	return df.pattern
}

// NewDescriptionFilterSet returns a new empty DescriptionFilterSet.
func NewDescriptionFilterSet(patterns ...string) (set DescriptionFilterSet, err error) {
	for _, pattern := range patterns {
		err = set.AppendPattern(pattern)
		if err != nil {
			return
		}
	}
	return
}

// Match returns true if the given string matches any of the
// description filters part of this DescriptionFilterSlice.
func (set DescriptionFilterSet) Match(str string) bool {
	for _, df := range set.filters {
		if df.Match(str) {
			return true
		}
	}
	return false
}

// AppendPattern appends a given pattern as a DescriptionFilter to this set.
func (set *DescriptionFilterSet) AppendPattern(pattern string) error {
	filter, err := NewDescriptionFilter(pattern)
	if err != nil {
		return fmt.Errorf("failed to append pattern: %v", err)
	}
	return set.Append(filter)
}

// Append appends a DescriptionFilter to this set.
func (set *DescriptionFilterSet) Append(filter *DescriptionFilter) error {
	for _, p := range set.filters {
		if p.pattern == filter.pattern {
			return fmt.Errorf("description filter set already contains pattern %s", filter.pattern)
		}
	}
	set.filters = append(set.filters, filter)
	return nil
}

// Difference returns the difference of this set and the other set,
// meaning it will return a new set containing all filters which are in this set that are not in the other set.
func (set DescriptionFilterSet) Difference(other DescriptionFilterSet) (c DescriptionFilterSet) {
	// copy internal slices and sort them
	a := DescriptionFilterSet{filters: make([]*DescriptionFilter, len(set.filters))}
	copy(a.filters, set.filters)
	sort.Sort(a)

	b := DescriptionFilterSet{filters: make([]*DescriptionFilter, len(other.filters))}
	copy(b.filters, other.filters)
	sort.Sort(b)

	lengthA, lengthB := a.Len(), b.Len()
	var indexA, indexB int
	for indexA < lengthA && indexB < lengthB {
		if a.filters[indexA].pattern == b.filters[indexB].pattern {
			indexA++
			indexB++
			continue
		}
		if a.filters[indexA].pattern < b.filters[indexB].pattern {
			// append from the first set
			c.Append(a.filters[indexA])
			indexA++
			continue
		}
		// only skip second set
		indexB++
	}
	// append all remaining ones
	for indexA < lengthA {
		c.Append(a.filters[indexA])
		indexA++
	}
	// sort our complement and return
	sort.Sort(c)
	return
}

// Length returns the length of the module identifier set.
func (set DescriptionFilterSet) Len() int {
	return len(set.filters)
}

// Less implemenets sort.Interface.Less
func (set DescriptionFilterSet) Less(i, j int) bool {
	return set.filters[i].pattern < set.filters[j].pattern
}

// Swap implemenets sort.Interface.Swap
func (set DescriptionFilterSet) Swap(i, j int) {
	set.filters[i], set.filters[j] = set.filters[j], set.filters[i]
}

// String returns the patterns used for this DescriptionFilterSlice as a CSV record.
func (set DescriptionFilterSet) String() string {
	n := len(set.filters)
	if n == 0 {
		return ""
	}
	ss := make([]string, n)
	for i, df := range set.filters {
		ss[i] = df.String()
	}
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	err := w.Write(ss)
	if err != nil {
		return ""
	}
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n")
}

// LoadString implements StringLoader.LoadString,
// allowing you to set the internal values of DescriptionFilterSet based
// on an earlier stringified DescriptionFilterSet.
func (set *DescriptionFilterSet) LoadString(str string) error {
	if len(str) == 0 {
		// set the empty set
		set.filters = nil
		return nil
	}

	// read the string as a SINGLE CSV RECORD
	r := csv.NewReader(strings.NewReader(str))
	record, err := r.Read()
	if err != nil {
		return fmt.Errorf("failed to read DescriptionFilterSet as CSV record: %v", err)
	}
	if record, err := r.Read(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("failed to read DescriptionFilterSet: "+
				"unexpected second CSV record found in given string: %s", strings.Join(record, ","))
		}
		return fmt.Errorf("failed to read DescriptionFilterSet: "+
			"unexpected error while checking for a second CSV record: %v", err)
	}

	// reset the set
	set.filters = nil
	// load all columns as separate patterns
	for i, pattern := range record {
		err = set.AppendPattern(pattern)
		if err != nil {
			return fmt.Errorf("failed to read DescriptionFilterSet: unexpected CSV record column #%d: %v", i, err)
		}
	}

	// string loaded as a single CSV record
	return nil
}
