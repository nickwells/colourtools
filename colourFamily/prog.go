package main

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/nickwells/col.mod/v6/col"
	"github.com/nickwells/col.mod/v6/colfmt"
	"github.com/nickwells/colour.mod/v2/colour"
	"github.com/nickwells/verbose.mod/verbose"
)

// prog holds program parameters and status
type prog struct {
	exitStatus int
	stack      *verbose.Stack
	// parameters
	families    []string
	showColours bool

	familyColours map[string][]string
}

// newProg returns a new Prog instance with the default values set
func newProg() *prog {
	allowedFamilies := colour.AllowedFamilies()
	p := prog{
		stack:         &verbose.Stack{},
		families:      slices.Collect(maps.Keys(allowedFamilies)),
		familyColours: make(map[string][]string),
	}

	return &p
}

// setExitStatus sets the exit status to the new value. It will not do this
// if the exit status has already been set to a non-zero value.
func (prog *prog) setExitStatus(es int) {
	if prog.exitStatus == 0 {
		prog.exitStatus = es
	}
}

// addColourNameCountVal adds the count of allowed colour names to the vals
// parameter and returns it. It will do nothing if the prog has a non-zero
// exitStatus. It will report any errors and set the prog's exitStatus.
func (prog *prog) addColourNameCountVal(vals *[]any, f colour.Family) {
	if prog.exitStatus != 0 {
		return
	}

	n, err := f.ColourNameCount()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		prog.setExitStatus(1)

		return
	}

	*vals = append(*vals, n)
}

// addDistinctColourCountVal adds the count of allowed colour names to the vals
// parameter and returns it. It will do nothing if the prog has a non-zero
// exitStatus. It will report any errors and set the prog's exitStatus.
func (prog *prog) addDistinctColourCountVal(vals *[]any, f colour.Family) {
	if prog.exitStatus != 0 {
		return
	}

	n, err := f.DistinctColourCount()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		prog.setExitStatus(1)

		return
	}

	*vals = append(*vals, n)
}

// addDescriptionVal adds the count of allowed colour names to the vals
// parameter and returns it. It will do nothing if the prog has a non-zero
// exitStatus. It will report any errors and set the prog's exitStatus.
func (prog *prog) addDescriptionVal(vals *[]any, f colour.Family) {
	if prog.exitStatus != 0 {
		return
	}

	desc, err := f.Description()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		prog.setExitStatus(1)

		return
	}

	*vals = append(*vals, desc)
}

// maxFamilyNameLen returns the length of the longest family name
func (prog *prog) maxFamilyNameLen() int {
	maxNameLen := 0
	for _, f := range prog.families {
		maxNameLen = max(maxNameLen, len(f))
	}

	return maxNameLen
}

// maxColourNameLen returns the length of the longest colour name
func (prog *prog) maxColourNameLen() int {
	maxNameLen := 0

	for _, cNames := range prog.familyColours {
		for _, c := range cNames {
			maxNameLen = max(maxNameLen, len(c))
		}
	}

	return maxNameLen
}

// colourNameReport generates a report showing the colour names for each
// family.
func (prog *prog) colourNameReport() {
	cols := []*col.Col{}

	for _, f := range prog.families {
		fFam := colour.Family(f)

		cNames, err := fFam.ColourNames()
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			prog.setExitStatus(1)

			return
		}

		slices.Sort(cNames)
		prog.familyColours[f] = cNames
	}

	cols = append(cols,
		col.New(&colfmt.String{W: prog.maxColourNameLen()}, "colours"))

	rpt := col.StdRpt(
		col.New(
			&colfmt.String{
				W:       prog.maxFamilyNameLen(),
				DupHdlr: colfmt.DupHdlr{SkipDups: true},
			},
			"family", "name"),
		cols...)

FamilyLoop:
	for _, f := range prog.families {
		if prog.showColours {
			for _, cn := range prog.familyColours[f] {
				vals := []any{}
				vals = append(vals, f, cn)

				if err := rpt.PrintRow(vals...); err != nil {
					fmt.Fprint(os.Stderr, err)
					break FamilyLoop
				}
			}
		}
	}
}

// standardReport prints the standard families report
func (prog *prog) standardReport() {
	const descriptionWidth = 50

	cols := []*col.Col{}
	cols = append(cols,
		col.New(&colfmt.Int{}, "colours", "named"),
		col.New(&colfmt.Int{}, "colours", "distinct"),
		col.New(&colfmt.WrappedString{W: descriptionWidth}, "description"),
	)

	rpt := col.StdRpt(
		col.New(
			&colfmt.String{
				W:       prog.maxFamilyNameLen(),
				DupHdlr: colfmt.DupHdlr{SkipDups: true},
			},
			"family", "name"),
		cols...)

	for _, f := range prog.families {
		fFam := colour.Family(f)
		vals := []any{}
		vals = append(vals, f)
		prog.addColourNameCountVal(&vals, fFam)
		prog.addDistinctColourCountVal(&vals, fFam)
		prog.addDescriptionVal(&vals, fFam)

		if prog.exitStatus != 0 {
			return
		}

		if err := rpt.PrintRow(vals...); err != nil {
			fmt.Fprint(os.Stderr, err)
			break
		}
	}
}

// run is the starting point for the program, it should be called from main()
// after the command-line parameters have been parsed. Use the setExitStatus
// method to record the exit status and then main can exit with that status.
func (prog *prog) run() {
	slices.Sort(prog.families)

	if prog.showColours {
		prog.colourNameReport()
	} else {
		prog.standardReport()
	}
}
