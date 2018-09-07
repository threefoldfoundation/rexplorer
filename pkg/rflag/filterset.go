package rflag

import (
	"github.com/spf13/pflag"
	"github.com/threefoldfoundation/rexplorer/pkg/types"
)

// DescriptionFilterSetFlagVar defines a DescriptionFilterSet flag with specified name, shorthand and usage string.
// The argument s points to a DescriptionFilterSet variable in which to store the compiled values of the multiple flags.
// The value of each argument will not try to be separated by comma, each value has to be defined as a seperate flag (using the same name).
func DescriptionFilterSetFlagVar(f *pflag.FlagSet, s *types.DescriptionFilterSet, name string, usage string) {
	f.Var(&descriptionFilterSetFlag{set: s}, name, usage)
}

// DescriptionFilterSetFlagVarP defines a DescriptionFilterSet flag with specified name, and usage string.
// The argument s points to a DescriptionFilterSet variable in which to store the compiled values of the multiple flags.
// The value of each argument will not try to be separated by comma, each value has to be defined as a seperate flag (using the same name or shorthand).
func DescriptionFilterSetFlagVarP(f *pflag.FlagSet, s *types.DescriptionFilterSet, name, shorthand string, usage string) {
	f.VarP(&descriptionFilterSetFlag{set: s}, name, shorthand, usage)
}

type descriptionFilterSetFlag struct {
	set     *types.DescriptionFilterSet
	changed bool
}

// Set implements pflag.Value.Set
func (flag *descriptionFilterSetFlag) Set(val string) error {
	if !flag.changed {
		var err error
		*flag.set, err = types.NewDescriptionFilterSet()
		if err != nil {
			return err
		}
		flag.changed = true
	}
	return flag.set.AppendPattern(val)
}

// Type implements pflag.Value.Type
func (flag *descriptionFilterSetFlag) Type() string {
	return "DescriptionFilterSetFlag"
}

// String implements pflag.Value.String
func (flag *descriptionFilterSetFlag) String() string {
	return flag.set.String()
}
