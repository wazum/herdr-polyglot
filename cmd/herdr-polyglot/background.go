package main

import (
	"os"

	"github.com/muesli/termenv"
)

// paneBackground asks the terminal what colour it is painted in. Inside a herdr
// popup that answer is the colour herdr painted, so the overlay can draw in the
// same shade instead of leaving a patch of its own. A terminal that stays silent
// gives an empty answer, and then nothing is painted that need not be.
func paneBackground() string {
	output := termenv.NewOutput(os.Stdout)

	background := output.BackgroundColor()
	if background == nil {
		return ""
	}

	hex := termenv.ConvertToRGB(background).Hex()
	if hex == "" || hex == "#000000" && !output.HasDarkBackground() {
		return ""
	}
	return hex
}
