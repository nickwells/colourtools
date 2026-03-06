package main

import (
	"fmt"
	"image/color" //nolint:misspell
	"regexp"
	"slices"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/colour.mod/v2/colour"
	"github.com/nickwells/coloursetter.mod/v2/coloursetter"
	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/groupsetter.mod/groupsetter"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v7/paction"
	"github.com/nickwells/param.mod/v7/param"
	"github.com/nickwells/param.mod/v7/psetter"
)

const (
	paramNameColour             = "colour"
	paramNameColoursNamedLike   = "colours-named-like"
	paramNameSimilarColour      = "colours-similar-to"
	paramNameColoursBetween     = "colours-between"
	paramNameColourCount        = "colour-count"
	paramNameColourfulContrast  = "colourful-contrast"
	paramNameText               = "text"
	paramNameLuminanceVariants  = "luminance-variants"
	paramNameSaturationVariants = "saturation-variants"
	paramNameInvertColour       = "invert-colour"
	paramNameComplementColour   = "complement-colour"
	paramNameSearchFamilies     = "families"
)

// makeColourGroupSetter creates the groupsetter.List for the colour
// parameters group. This will populate a slice of colourListEntry's.
func makeColourGroupSetter(prog *prog) *groupsetter.List[colourListEntry] {
	const (
		paramNameBackground       = "background"
		paramNameForegroundColour = "foreground-colour"
		paramNameText             = "text"
	)

	gSetter := groupsetter.NewList(
		&prog.colourList,
		groupsetter.SetInterimValueInitFunc(
			func(v *colourListEntry) error {
				v.text = prog.dfltText

				return nil
			}))

	gSetter.AddByPosParam(paramNameBackground,
		coloursetter.NamedColour{Value: &gSetter.InterimVal.bg},
		"set the value of the background colour to be displayed.",
	)

	gSetter.AddByNameParam(paramNameForegroundColour,
		coloursetter.NamedColour{Value: &gSetter.InterimVal.fg},
		"set the value of the foreground colour to use."+
			" If no foreground colour is given then"+
			" a contrasting colour will be generated",
		param.AltNames("foreground-color", //nolint:misspell
			"foreground",
			"fg",
			"fg-colour",
			"fg-color", //nolint:misspell
		),
		param.PostAction(
			paction.SetVal(&gSetter.InterimVal.fgGiven, true)),
	)

	gSetter.AddByNameParam(paramNameText,
		psetter.String[string]{Value: &gSetter.InterimVal.text},
		"set the value of the text to be displayed",
		param.AltNames("txt"),
	)

	gSetter.AddFinalCheck(func() error {
		bgCol := gSetter.InterimVal.bg.Colour()
		if !gSetter.InterimVal.fgGiven {
			gSetter.InterimVal.fg = colour.MakeNamedColour(
				prog.dfltFGColourName,
				prog.contrastColour(bgCol))
		}

		return nil
	})

	return gSetter
}

// makeUniqueNamedColours takes a slice of NamedColours and replaces entries
// having the same colour with a new entry having that colour but with the
// name expanded to show all the alternative names.
func makeUniqueNamedColours(nc []colour.NamedColour) []colour.NamedColour {
	uniqueNC := []colour.NamedColour{}

	uniqueMap := map[color.RGBA][]string{} //nolint:misspell

	for _, v := range nc {
		uniqueMap[v.Colour()] = append(uniqueMap[v.Colour()], v.Name())
	}

	for c, names := range uniqueMap {
		names = colour.StripAliases(names)

		uniqueNC = append(uniqueNC,
			colour.MakeNamedColour(
				english.JoinQuoted(names, ", ", " or "),
				c))
	}

	slices.SortFunc(uniqueNC, colour.NamedColourCompare)

	return uniqueNC
}

// regexpPostAction generates a param.ActionFunc which will populate the
// prog.colourList with colours whose names match the regular expression.
func regexpPostAction(prog *prog, re **regexp.Regexp) param.ActionFunc {
	return func(_ location.L, _ *param.BaseParam, _ []string) error {
		namedColours, err := colour.ColoursMatchingByRegexp(
			prog.families, *re)
		if err != nil {
			return err
		}

		namedColours = makeUniqueNamedColours(namedColours)
		slices.SortFunc(namedColours, colour.NamedColourCompare)

		prog.addToColourList(namedColours...)

		return nil
	}
}

// similarColourPostAction generates a param.ActionFunc which will populate
// the prog.colourList with colours close to the target colour.
func similarColourPostAction(
	prog *prog, targetC *color.RGBA, //nolint:misspell
) param.ActionFunc {
	return func(_ location.L, _ *param.BaseParam, _ []string) error {
		familyColours, err := prog.families.ClosestN(*targetC, prog.colourCount)
		if err != nil {
			return err
		}

		namedColours := []colour.NamedColour{}
		for _, fc := range familyColours {
			namedColours = append(namedColours,
				colour.MakeDistinctNamedColoursFromFamilyColour(fc)...)
		}

		namedColours = makeUniqueNamedColours(namedColours)
		slices.SortFunc(namedColours, colour.NamedColourCompare)

		prog.addToColourList(namedColours...)

		return nil
	}
}

// betweenColoursPostAction generates a param.ActionFunc which will populate
// the prog.colourList with colours close to the target colour.
func betweenColoursPostAction(
	prog *prog, lowerColour, upperColour *color.RGBA, //nolint:misspell
) param.ActionFunc {
	return func(_ location.L, _ *param.BaseParam, _ []string) error {
		cList, err := colour.MakeColoursBetween(
			prog.colourCount, *lowerColour, *upperColour)
		if err != nil {
			return err
		}

		namedColours := []colour.NamedColour{}
		for _, c := range cList {
			namedColours = append(namedColours,
				colour.MakeNamedColour("generated colour", c))
		}

		prog.addToColourList(namedColours...)

		return nil
	}
}

// makeLuminanceVals returns a slice of luminance values from which to
// generate colours. Note that the generated luminance values range between 0
// and 1 but exclude the extremes as the resultant generated colours will
// always be either black or white.
func makeLuminanceVals(prog *prog) []float64 {
	lumVals := []float64{}

	interval := 1.0 / float64(prog.colourCount+1)
	lv := 1.0

	for range prog.colourCount {
		lv -= interval
		lumVals = append(lumVals, lv)

		if lv <= 0 {
			break
		}
	}

	return lumVals
}

// lumColoursPostAction genmerates a param.ActionFunc which will
// populate the prog.colourList with a range of colours similar to the
// supplied colour but with different luminances.
func lumColoursPostAction(prog *prog, nc *colour.NamedColour) param.ActionFunc {
	return func(_ location.L, _ *param.BaseParam, _ []string) error {
		lumVals := makeLuminanceVals(prog)
		namedColours := []colour.NamedColour{}
		c := nc.Colour()

		for _, lv := range lumVals {
			lumC, err := colour.Luminance(c, lv)
			if err != nil {
				return fmt.Errorf(
					"could not generate the luminance value: %w", err)
			}

			name := fmt.Sprintf("generated colour (luminance: %.3f)", lv)

			namedColours = append(namedColours,
				colour.MakeNamedColour(name, lumC))
		}

		prog.addToColourList(namedColours...)

		return nil
	}
}

// makeSaturationVals returns a slice of saturation values from which to
// generate colours. Note that the generated saturation values range between 0
// and 1.
func makeSaturationVals(prog *prog) []float64 {
	lumVals := []float64{}

	interval := 1.0 / float64(prog.colourCount+1)
	lv := 1.0

	for range prog.colourCount {
		lumVals = append(lumVals, lv)

		lv -= interval
		if lv < 0 {
			break
		}
	}

	return lumVals
}

// satColoursPostAction genmerates a param.ActionFunc which will
// populate the prog.colourList with a range of colours similar to the
// supplied colour but with different saturations.
func satColoursPostAction(prog *prog, nc *colour.NamedColour) param.ActionFunc {
	return func(_ location.L, _ *param.BaseParam, _ []string) error {
		lumVals := makeSaturationVals(prog)
		namedColours := []colour.NamedColour{}

		c := nc.Colour()
		for _, lv := range lumVals {
			lumC, err := colour.Saturation(c, lv)
			if err != nil {
				return fmt.Errorf(
					"could not generate the saturation value: %w", err)
			}

			name := fmt.Sprintf("generated colour (saturation: %.3f)", lv)

			namedColours = append(namedColours,
				colour.MakeNamedColour(name, lumC))
		}

		prog.addToColourList(namedColours...)

		return nil
	}
}

// invertColourPostAction generates a param.ActionFunc which will add to the
// prog.colourList a colour which is the inverse of the supplied colour.
func invertColourPostAction(
	prog *prog, nc *colour.NamedColour,
) param.ActionFunc {
	return func(_ location.L, _ *param.BaseParam, _ []string) error {
		invert := colour.Invert(nc.Colour())

		name := fmt.Sprintf("inverse of %q", nc.Name())

		prog.addToColourList(colour.MakeNamedColour(name, invert))

		return nil
	}
}

// complementColourPostAction generates a param.ActionFunc which will add to
// the prog.colourList a colour which is the complement of the supplied
// colour.
func complementColourPostAction(
	prog *prog, nc *colour.NamedColour,
) param.ActionFunc {
	return func(_ location.L, _ *param.BaseParam, _ []string) error {
		complement := colour.Complement(nc.Colour())

		name := fmt.Sprintf("complement of %q", nc.Name())

		prog.addToColourList(colour.MakeNamedColour(name, complement))

		return nil
	}
}

// addParams adds the parameters for this program
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		var colourParamCounter paction.Counter

		counterActionFunc := colourParamCounter.MakeActionFunc()

		colourParams := []string{
			paramNameColour,
			paramNameColoursNamedLike,
			paramNameSimilarColour,
			paramNameColoursBetween,
		}

		gSetter := makeColourGroupSetter(prog)

		ps.Add(paramNameColour, gSetter,
			"the colour to be displayed, if no foreground colour"+
				" is given then a contrasting colour will be chosen.",
			param.AltNames(
				"c",
				"color", //nolint:misspell
			),
			param.PostAction(counterActionFunc),
			param.SeeAlso(colourParams...),
		)

		var colourRE *regexp.Regexp
		ps.Add(paramNameColoursNamedLike, psetter.Regexp{
			Value: &colourRE,
		},
			"a Regular Expression for selecting colours by name."+
				" A new entry in the list of colours will be"+
				" generated for each matching name.",
			param.AltNames("named-like"),
			param.PostAction(regexpPostAction(prog, &colourRE)),
			param.PostAction(counterActionFunc),
			param.SeeAlso(colourParams...),
		)

		var simTargetColour color.RGBA //nolint:misspell
		ps.Add(paramNameSimilarColour, coloursetter.RGB{
			Value: &simTargetColour,
		},
			"a target colour to find colours similar to this."+
				" A new entry in the list of colours will be"+
				" generated for each similar colour.",
			param.AltNames(
				"similar-to",
				"close-to",
				"like",
			),
			param.PostAction(similarColourPostAction(prog, &simTargetColour)),
			param.PostAction(counterActionFunc),
			param.SeeAlso(colourParams...),
		)

		var lumTargetColour colour.NamedColour
		ps.Add(paramNameLuminanceVariants, coloursetter.NamedColour{
			Value: &lumTargetColour,
		},
			"a target colour from which to generate"+
				" a range of colours with different luminances.",
			param.PostAction(lumColoursPostAction(prog, &lumTargetColour)),
			param.PostAction(counterActionFunc),
			param.SeeAlso(colourParams...),
		)

		var satTargetColour colour.NamedColour
		ps.Add(paramNameSaturationVariants, coloursetter.NamedColour{
			Value: &satTargetColour,
		},
			"a target colour from which to generate"+
				" a range of colours with different saturations.",
			param.PostAction(satColoursPostAction(prog, &satTargetColour)),
			param.PostAction(counterActionFunc),
			param.SeeAlso(colourParams...),
		)

		var invTargetColour colour.NamedColour
		ps.Add(paramNameInvertColour, coloursetter.NamedColour{
			Value: &invTargetColour,
		},
			"a target colour from which to generate its inverse.",
			param.AltNames("invert",
				"colour-invert",
				"color-invert", "invert-color", //nolint:misspell
			),
			param.PostAction(invertColourPostAction(prog, &invTargetColour)),
			param.PostAction(counterActionFunc),
			param.SeeAlso(colourParams...),
		)

		var compTargetColour colour.NamedColour
		ps.Add(paramNameComplementColour, coloursetter.NamedColour{
			Value: &compTargetColour,
		},
			"a target colour from which to generate its complement.",
			param.AltNames("complement",
				"colour-complement",
				"color-complement", "complement-color", //nolint:misspell
			),
			param.PostAction(complementColourPostAction(prog, &compTargetColour)),
			param.PostAction(counterActionFunc),
			param.SeeAlso(colourParams...),
		)

		var lowerColour, upperColour color.RGBA //nolint:misspell
		ps.Add(paramNameColoursBetween, coloursetter.RGBPair{
			Value1: &lowerColour,
			Value2: &upperColour,
		},
			"a pair of colours to generate colours between."+
				" A new entry in the list of colours will be"+
				" generated for each generated colour.",
			param.AltNames("between"),
			param.PostAction(betweenColoursPostAction(
				prog, &lowerColour, &upperColour)),
			param.PostAction(counterActionFunc),
			param.SeeAlso(colourParams...),
		)

		ps.Add(paramNameText, psetter.String[string]{
			Value: &prog.dfltText,
		},
			"the default text to use for any subsequent colours.",
			param.AltNames("txt", "default-text"),
		)

		const maxColourCount = 100
		ps.Add(paramNameColourCount,
			psetter.Int[int]{
				Value: &prog.colourCount,
				Checks: []check.ValCk[int]{
					check.ValGE(1),
					check.ValLE(maxColourCount),
				},
			},
			"the maximum number of colours to generate when finding similar"+
				" colours or generating colours."+
				" For this to take effect this parameter"+
				" must be given before the similar colours parameter."+
				"\n\n"+
				"Note that the number of resulting colours may be fewer"+
				" than this (even if there are more possible colours)"+
				" since identical colours in other colour families are"+
				" merged after the set of colours is truncated to this"+
				" maximum number",
			param.SeeAlso(colourParams...),
		)

		ps.Add(paramNameSearchFamilies,
			coloursetter.Families{
				Value: &prog.families,
			},
			"the families to search when looking for"+
				" colours matching by name or proximity")

		ps.Add(paramNameColourfulContrast, psetter.Bool{
			Value: &prog.colourfulContrast,
		},
			"signals that the alternative algorithm for auto-generating"+
				" contrasting colours should be used",
		)

		ps.AddFinalCheck(func() error {
			if colourParamCounter.Count() == 0 {
				return fmt.Errorf(
					"you must give at least one of these parameters: %s",
					english.JoinQuoted(colourParams, ", ", " or "))
			}

			return nil
		})

		return nil
	}
}
