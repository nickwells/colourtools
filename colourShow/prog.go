package main

import (
	"fmt"
	"html/template"
	"image/color" //nolint:misspell
	"math"
	"net/http"
	"os"

	"github.com/nickwells/colour.mod/v2/colour"
	"github.com/nickwells/colourtools/internal/web"
	"github.com/nickwells/verbose.mod/verbose"
)

const (
	ExitStatusServerStartup   = 1
	ExitStatusTemplateFailure = 2
)

// colourListEntry records the values needed to generate a colour display
//
//nolint:misspell
type colourListEntry struct {
	// fg is the foreground colour to be used
	fg colour.NamedColour
	// fgGiven is a flag indicating whether or not the foreground colour was
	// given explicitly. If not then the foreground colour will be generated
	fgGiven bool
	// bg is the background colour
	bg colour.NamedColour
	// text is the text to use (it will be coloured using the foreground
	// colour)
	text string
}

// String returns a string value representing the colourListEntry value.
func (cle colourListEntry) String() string {
	rval := ""

	rval += fmt.Sprintf("  bg: %+v (%q)\n", cle.bg, cle.bg.Name())
	rval += fmt.Sprintf("  fg: %+v (%s)\n", cle.fg, cle.fg.Name())
	rval += fmt.Sprintf("text: %q\n", cle.text)

	return rval
}

// prog holds program parameters and status
type prog struct {
	exitStatus int
	stack      *verbose.Stack
	// parameters

	// colourList is the collection of background and foreground colours and
	// the text to use for each row in the colour table on the generated web
	// page
	colourList []colourListEntry
	// colourCount is the maximum number of colours to generate when finding
	// colours close to the supplied colour
	colourCount int
	// dfltText is the text to use for the following colours
	dfltText string
	// use the algorithm that generates more colourful contrast colours
	colourfulContrast bool
	// the name to give a generated foreground colour
	dfltFGColourName string
	// the families to search when looking for colourList
	families colour.Families
}

// newProg returns a new Prog instance with the default values set
func newProg() *prog {
	const dfltColourCount = 5

	return &prog{
		stack:            &verbose.Stack{},
		colourCount:      dfltColourCount,
		dfltText:         "Hello, World!",
		dfltFGColourName: "generated contrasting colour",
	}
}

// setExitStatus sets the exit status to the new value. It will not do this
// if the exit status has already been set to a non-zero value.
func (prog *prog) setExitStatus(es int) {
	if prog.exitStatus == 0 {
		prog.exitStatus = es
	}
}

// contrastColour returns a generated contrasting colour. It respects the
// setting of the colourfulContrast flag when choosing the algorithm to use
// to generate the colour.
func (prog *prog) contrastColour(c color.RGBA) color.RGBA { //nolint:misspell
	generator := colour.Contrast
	if prog.colourfulContrast {
		generator = colour.ContrastColourful
	}

	return generator(c)
}

// addToColourList adds the named colours to the colourList
func (prog *prog) addToColourList(namedColours ...colour.NamedColour) {
	for _, nc := range namedColours {
		prog.colourList = append(prog.colourList,
			colourListEntry{
				text: prog.dfltText,
				bg:   nc,
				fg: colour.MakeNamedColour(
					prog.dfltFGColourName,
					prog.contrastColour(nc.Colour()),
				),
			})
	}
}

// run is the starting point for the program, it should be called from main()
// after the command-line parameters have been parsed. Use the setExitStatus
// method to record the exit status and then main can exit with that status.
func (prog *prog) run() {
	s, err := web.MakeOneTimeServer(prog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't make the one-time server: %s\n", err)
		prog.setExitStatus(ExitStatusServerStartup)

		return
	}

	err = s.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't start the one-time server: %s\n", err)
		prog.setExitStatus(ExitStatusServerStartup)

		return
	}
}

const (
	pgStart = `<!DOCTYPE html>
<html>
<head>
    <title>ColourShow</title>
</head>
<body>
    <table>
        <tr>
            <th>Foreground<br>Colour</th>
            <th>Background<br>Colour</th>
            <th>FG to BG<br>Distance</th>
            <th>Example</th>
            <th>Colours<br>Reversed</th>
        </tr>`

	pgEnd = `</table></body></html>`

	//nolint:misspell
	divTemplate = `{{define "div"}}
        <tr>
            <td>
                <div style="display: flex;
                            width: 100%;
                            min-height: 200px;
                            align-items: center;
                            justify-content: center;
                            padding:10px;">
                    {{.FGColourName}}
                    <br>
                    <br>
                    {{.FGColourRGB}}
                </div>
            </td>
            <td>
                <div style="display: flex;
                            width: 100%;
                            min-height: 200px;
                            align-items: center;
                            justify-content: center;
                            padding:10px;">
                    {{.BGColourName}}
                    <br>
                    <br>
                    {{.BGColourRGB}}
                </div>
            </td>
            <td>
                <div style="display: flex;
                            width: 100%;
                            min-height: 200px;
                            align-items: center;
                            justify-content: center;
                            padding:10px;">
                    <br>
                    <br>
                    {{.Dist}}
                </div>
            </td>
            <td>
                <div style="display: flex;
                            width: 100%;
                            min-height: 200px;
                            align-items: center;
                            justify-content: center;
                            padding:10px;
                            font-size: 2.5rem;
                            background-color: {{.BgStr}};
                            color: {{.FgStr}};">
                    {{.Text}}
                </div>
            </td>
            <td>
                <div style="display: flex;
                            width: 100%;
                            min-height: 200px;
                            align-items: center;
                            justify-content: center;
                            padding:10px;
                            font-size: 2.5rem;
                            background-color: {{.FgStr}};
                            color: {{.BgStr}};">
                    {{.Text}}
                </div>
            </td>
        </tr>
{{end}}`
)

type div struct {
	BGColourName string
	BGColourRGB  string
	FGColourName string
	FGColourRGB  string
	BgStr        template.CSS
	FgStr        template.CSS
	Text         string
	Dist         string
}

// dist returns the Euclidean distance between the two colours in the RGB
// colour cube.
func dist(c1, c2 color.RGBA) float64 { //nolint:misspell
	rDiff := int(c1.R) - int(c2.R)
	gDiff := int(c1.G) - int(c2.G)
	bDiff := int(c1.B) - int(c2.B)

	rDiff2 := rDiff * rDiff
	gDiff2 := gDiff * gDiff
	bDiff2 := bDiff * bDiff

	return math.Sqrt(float64(rDiff2 + gDiff2 + bDiff2))
}

// makeDiv generates the div struct populating the fields from the
// colourListEntry
func makeDiv(c colourListEntry) div {
	const (
		rgbFmt  = "rgb(%d,%d,%d)"
		nameFmt = "#%02x%02x%02x"
	)

	d := div{
		Text: c.text,
		Dist: fmt.Sprintf("%.2f", dist(c.bg.Colour(), c.fg.Colour())),
	}

	bgCol := c.bg.Colour()
	fgCol := c.fg.Colour()
	d.BgStr = template.CSS(fmt.Sprintf(rgbFmt, //nolint:gosec
		bgCol.R, bgCol.G, bgCol.B))
	d.FgStr = template.CSS(fmt.Sprintf(rgbFmt, //nolint:gosec
		fgCol.R, fgCol.G, fgCol.B))

	d.BGColourName = c.bg.Name()
	d.BGColourRGB = fmt.Sprintf(nameFmt, bgCol.R, bgCol.G, bgCol.B)

	d.FGColourName = c.fg.Name()

	d.FGColourRGB = fmt.Sprintf(nameFmt, fgCol.R, fgCol.G, fgCol.B)

	return d
}

// ServeHTTP sends the web page
func (prog *prog) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	t, err := template.New("div").Parse(divTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot parse the div template: %v\n", err)
		prog.setExitStatus(ExitStatusTemplateFailure)

		return
	}

	fmt.Fprint(rw, pgStart)

	for _, c := range prog.colourList {
		d := makeDiv(c)

		err = t.ExecuteTemplate(rw, "div", d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot execute the div template: %v\n", err)
			prog.setExitStatus(ExitStatusTemplateFailure)

			break
		}
	}

	fmt.Fprint(rw, pgEnd)
}
