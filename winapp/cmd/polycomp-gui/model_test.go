package main

import (
	"testing"

	"polycomp/winapp/internal/gtxc"
)

// The exact page the sign displayed on 2026-10-12 12:28:59 (two tall rows, centred, flashing green arrow).
func TestMessageFramesMatchLivePage(t *testing.T) {
	p := NewPageModel()
	p.Layout = int(gtxc.TwoTall)
	p.Rows[0] = RowModel{Text: "Cappuccino R35", Align: int(gtxc.AlignCentre)}
	p.Rows[1] = RowModel{Text: "<--", Colour: int(gtxc.Green), Flash: true, Align: int(gtxc.AlignCentre)}
	frames, err := Message{Pages: []PageModel{p}}.Frames(4, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := "00 04 00 03 C8 30 30 31 E2 C1 83 20 43 61 70 70 75 63 63 69 6E 6F 20 52 33 35" // header + centred row 1
	if got := gtxc.Hex(frames[0][:26]); got != want {
		t.Fatalf("got %s\nwant %s", got, want)
	}
	tail := gtxc.Hex(frames[0][len(frames[0])-9:])
	if tail != "1C 47 1C 46 3C 2D 2D 04 2C" {
		t.Fatalf("tail %s", tail)
	}
}

func TestSequenceSetsMorePagesBit(t *testing.T) {
	a, b := NewPageModel(), NewPageModel()
	a.Rows[0].Text, b.Rows[0].Text = "one", "two"
	frames, _ := Message{Pages: []PageModel{a, b}}.Frames(0, 0)
	if frames[0][4] != 0xCC || frames[1][4] != 0xC8 || string(frames[1][5:8]) != "002" {
		t.Fatalf("flags %02X %02X page %q", frames[0][4], frames[1][4], frames[1][5:8])
	}
}
