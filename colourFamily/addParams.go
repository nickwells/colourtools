package main

import (
	"github.com/nickwells/colour.mod/v2/colour"
	"github.com/nickwells/param.mod/v7/param"
	"github.com/nickwells/param.mod/v7/psetter"
)

const (
	paramNameFamilies    = "families"
	paramNameShowColours = "show-colours"
)

// addParams adds the parameters for this program
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add(paramNameFamilies,
			psetter.EnumList[string]{
				Value:       &prog.families,
				AllowedVals: colour.AllowedFamilies(),
			},
			"a list of colour families to display details for",
			param.AltNames("family", "f"))
		ps.Add(paramNameShowColours,
			psetter.Bool{
				Value: &prog.showColours,
			},
			"If set all the colours for the given families will be listed",
			param.AltNames(
				"colours", "colors", //nolint:misspell
				"show-colors", "show-col")) //nolint:misspell

		return nil
	}
}
