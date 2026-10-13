package gtxc

import "fmt"

// Row geometry of the 192x32 AMP2008 sign, verified live.
const (
	RowChars    = 32 // characters on one 8 px row
	TallVisible = 16 // characters shown on a joined 16 px row pair
	PairChars   = 64 // text a joined pair still consumes
)

type Colour int

const (
	Red Colour = iota // the sign's default
	Green
	Yellow
)

var colourNames = []string{"Red", "Green", "Yellow"}
var colourCodes = []byte{'R', 'G', 'Y'}

func (c Colour) String() string { return colourNames[c] }
func ColourNames() []string     { return append([]string(nil), colourNames...) }

type Align int

const (
	AlignLeft Align = iota
	AlignCentre
	AlignRight
)

var alignNames = []string{"Left", "Centre", "Right"}

func (a Align) String() string { return alignNames[a] }
func AlignNames() []string     { return append([]string(nil), alignNames...) }

// LayoutMode selects which row pairs are joined into tall rows.
type LayoutMode int

const (
	FourSmall  LayoutMode = iota // 4 rows x 32 chars
	TallTop                      // 1 tall (16) + 2 small (32)
	TallBottom                   // 2 small + 1 tall
	TwoTall                      // 2 tall rows
)

var layoutNames = []string{"4 small rows", "Big top row + 2 small rows", "2 small rows + big bottom row", "2 big rows"}

func (m LayoutMode) String() string { return layoutNames[m] }
func LayoutNames() []string         { return append([]string(nil), layoutNames...) }

// Rows returns (visible width, consumed width) for each visual row of the mode.
func (m LayoutMode) Rows() [][2]int {
	tall := [2]int{TallVisible, PairChars}
	small := [2]int{RowChars, RowChars}
	switch m {
	case TallTop:
		return [][2]int{tall, small, small}
	case TallBottom:
		return [][2]int{small, small, tall}
	case TwoTall:
		return [][2]int{tall, tall}
	}
	return [][2]int{small, small, small, small}
}

// BoldBits returns the PAGE STATUS join bits for the mode.
func (m LayoutMode) BoldBits() byte {
	switch m {
	case TallTop:
		return 0b01
	case TallBottom:
		return 0b10
	case TwoTall:
		return 0b11
	}
	return 0
}

// Row is one visual row as the user edits it.
type Row struct {
	Text   string
	Colour Colour
	Flash  bool
	Wide   bool // 0x1C E: double width, same height
	Align  Align
}

// Cells is the width the row's text occupies (wide characters count double).
func (r Row) Cells() int {
	n := len([]rune(r.Text))
	if r.Wide {
		n *= 2
	}
	return n
}

// Layout lays the rows out as the sign's text stream: rows are filled in order, a row is padded
// with spaces to the width it consumes, formatting is reset with 0x1C D at the end of a formatted
// row so padding and the next row start clean. The last row is not padded.
func Layout(rows []Row, mode LayoutMode) ([]byte, error) {
	geo := mode.Rows()
	if len(rows) > len(geo) {
		return nil, fmt.Errorf("%d rows given but %q has %d", len(rows), mode, len(geo))
	}
	var out []byte
	for i, r := range rows {
		width, consumed := geo[i][0], geo[i][1]
		cells := r.Cells()
		if cells > width {
			return nil, fmt.Errorf("row %d is %d cells wide, the row holds %d: %q", i+1, cells, width, r.Text)
		}
		enc, err := EncodeCP437(r.Text)
		if err != nil {
			return nil, fmt.Errorf("row %d: %v", i+1, err)
		}
		lead := 0
		switch r.Align {
		case AlignCentre:
			lead = (width - cells) / 2
		case AlignRight:
			lead = width - cells
		}
		out = append(out, spaces(lead)...)
		formatted := r.Colour != Red || r.Flash || r.Wide
		if r.Colour != Red {
			out = append(out, Fmt(colourCodes[r.Colour])...)
		}
		if r.Flash {
			out = append(out, Fmt('F')...)
		}
		if r.Wide {
			out = append(out, Fmt('E')...)
		}
		out = append(out, enc...)
		last := i == len(rows)-1
		if formatted && !last {
			out = append(out, Fmt('D')...)
		}
		if !last {
			out = append(out, spaces(consumed-lead-cells)...)
		}
	}
	return out, nil
}

func spaces(n int) []byte {
	if n <= 0 {
		return nil
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return b
}
