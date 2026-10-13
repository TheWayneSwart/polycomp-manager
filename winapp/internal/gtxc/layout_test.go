package gtxc

import (
	"bytes"
	"strings"
	"testing"
)

func pad(n int) string { return strings.Repeat(" ", n) }

func TestFourSmallRowsPaddedTo32(t *testing.T) {
	out, err := Layout([]Row{{Text: "ROW ONE"}, {Text: "ROW TWO"}, {Text: "ROW THREE"}, {Text: "ROW FOUR"}}, FourSmall)
	if err != nil {
		t.Fatal(err)
	}
	want := "ROW ONE" + pad(25) + "ROW TWO" + pad(25) + "ROW THREE" + pad(23) + "ROW FOUR"
	if string(out) != want {
		t.Fatalf("got %q", out)
	}
}

// Verified live: 'HELLO WAYNE' padded to 64 put ROW THREE / ROW FOUR on rows 3 and 4.
func TestTallTopConsumes64(t *testing.T) {
	out, _ := Layout([]Row{{Text: "HELLO WAYNE"}, {Text: "ROW THREE"}, {Text: "ROW FOUR"}}, TallTop)
	want := "HELLO WAYNE" + pad(53) + "ROW THREE" + pad(23) + "ROW FOUR"
	if string(out) != want {
		t.Fatalf("got %q", out)
	}
	if TallTop.BoldBits() != 1 || TwoTall.BoldBits() != 3 {
		t.Fatal("bold bits")
	}
}

func TestCentreAndRight(t *testing.T) {
	out, _ := Layout([]Row{{Text: "Cappuccino R35", Align: AlignCentre}, {Text: "Bean&Bite", Colour: Green, Align: AlignCentre}}, TwoTall)
	want := append([]byte(pad(1)+"Cappuccino R35"+pad(49)+pad(3)), 0x1C, 'G')
	want = append(want, []byte("Bean&Bite")...)
	if !bytes.Equal(out, want) {
		t.Fatalf("got %q", out)
	}
	out, _ = Layout([]Row{{Text: "ab", Align: AlignRight}, {Text: "x"}}, FourSmall)
	if string(out) != pad(30)+"ab"+"x" {
		t.Fatalf("got %q", out)
	}
}

func TestWideCountsDoubleAndResetsBeforePadding(t *testing.T) {
	out, _ := Layout([]Row{{Text: "Cappuccino", Wide: true, Align: AlignCentre}, {Text: "x"}}, FourSmall)
	want := append([]byte(pad(6)), 0x1C, 'E')
	want = append(want, []byte("Cappuccino")...)
	want = append(want, 0x1C, 'D')
	want = append(want, []byte(pad(6)+"x")...)
	if !bytes.Equal(out, want) {
		t.Fatalf("got %q", out)
	}
	if _, err := Layout([]Row{{Text: strings.Repeat("x", 17), Wide: true}}, FourSmall); err == nil {
		t.Fatal("34 cells accepted on a 32 row")
	}
}

func TestFlashAndTooManyRows(t *testing.T) {
	out, _ := Layout([]Row{{Text: "<--", Colour: Green, Flash: true}}, TwoTall)
	if !bytes.Equal(out, []byte{0x1C, 'G', 0x1C, 'F', '<', '-', '-'}) {
		t.Fatalf("got %q", out)
	}
	if _, err := Layout(make([]Row, 3), TwoTall); err == nil {
		t.Fatal("3 rows accepted in two-tall mode")
	}
}
