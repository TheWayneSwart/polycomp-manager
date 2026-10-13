//go:build windows

package main

import (
	"github.com/lxn/walk"

	"polycomp/winapp/internal/gtxc"
)

// Board preview: 192x32 LED matrix drawn at 4x, one character per cell (6x8 px small, 12x16 tall).
const scale = 4

var (
	ledBlack  = walk.RGB(12, 12, 12)
	ledRed    = walk.RGB(255, 48, 32)
	ledGreen  = walk.RGB(48, 255, 48)
	ledYellow = walk.RGB(255, 230, 40)
	ledDim    = walk.RGB(60, 60, 60)
)

func colourOf(c gtxc.Colour) walk.Color {
	switch c {
	case gtxc.Green:
		return ledGreen
	case gtxc.Yellow:
		return ledYellow
	}
	return ledRed
}

type preview struct {
	widget    *walk.CustomWidget
	smallFont *walk.Font
	tallFont  *walk.Font
	page      func() PageModel // current editor state
}

func (p *preview) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	bg, err := walk.NewSolidColorBrush(ledBlack)
	if err != nil {
		return err
	}
	defer bg.Dispose()
	w, h := 192*scale, 32*scale
	canvas.FillRectangle(bg, walk.Rectangle{X: 0, Y: 0, Width: w, Height: h})

	// faint module grid: 6 P10 modules of 32x16 px
	grid, _ := walk.NewCosmeticPen(walk.PenSolid, ledDim)
	defer grid.Dispose()
	for x := 32; x < 192; x += 32 {
		canvas.DrawLine(grid, walk.Point{X: x * scale, Y: 0}, walk.Point{X: x * scale, Y: h})
	}
	canvas.DrawLine(grid, walk.Point{X: 0, Y: 16 * scale}, walk.Point{X: w, Y: 16 * scale})

	pm := p.page()
	mode := gtxc.LayoutMode(pm.Layout)
	y := 0
	for i, geo := range mode.Rows() {
		tall := geo[0] == gtxc.TallVisible
		cellW, cellH, font := 6*scale, 8*scale, p.smallFont
		if tall {
			cellW, cellH, font = 12*scale, 16*scale, p.tallFont
		}
		r := pm.Rows[i]
		cells := len([]rune(r.Text))
		if r.Wide {
			cells *= 2
		}
		lead := 0
		switch gtxc.Align(r.Align) {
		case gtxc.AlignCentre:
			lead = (geo[0] - cells) / 2
		case gtxc.AlignRight:
			lead = geo[0] - cells
		}
		if lead < 0 {
			lead = 0
		}
		col := colourOf(gtxc.Colour(r.Colour))
		x := lead * cellW
		step := cellW
		if r.Wide {
			step = 2 * cellW
		}
		for _, ch := range r.Text {
			if x+step > w {
				break
			}
			rect := walk.Rectangle{X: x, Y: y, Width: step, Height: cellH}
			canvas.DrawText(string(ch), font, col, rect, walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
			x += step
		}
		y += cellH
	}
	return nil
}
