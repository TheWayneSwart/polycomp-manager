package gtxc

import (
	"bytes"
	"testing"
	"time"
)

// Vendor BASIC sample: 1-line sign, address 2, page 001, text "CNT = 100"
func TestVendorSingleLineExample(t *testing.T) {
	p := Page{Number: 1, Tempo: 0xE2, Function: 0xC1, Status: 0xC0, Text: []byte("CNT = 100")}
	f, err := BuildFrame(p, 1, 2, 0xC0)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte{0x00, 0x01, 0x02, 0x03, 0xC0, '0', '0', '1', 0xE2, 0xC1, 0xC0}, []byte("CNT = 100")...)
	want = append(want, 0x04)
	want = append(want, Checksum(want))
	if !bytes.Equal(f, want) {
		t.Fatalf("got %s want %s", Hex(f), Hex(want))
	}
}

// Frame ACKed by the real sign on 2026-10-12 11:29:22.
func TestLiveAckedFrame(t *testing.T) {
	p := Page{Number: 1, Tempo: 0xE2, Function: 0xC1, Status: 0xC0, Text: []byte("HELLO WAYNE 2")}
	f, _ := BuildFrame(p, 0, 0, Serst(false, true, false))
	if Hex(f) != "00 00 00 03 C8 30 30 31 E2 C1 C0 48 45 4C 4C 4F 20 57 41 59 4E 45 20 32 04 29" {
		t.Fatalf("got %s", Hex(f))
	}
}

func TestFlagBuilders(t *testing.T) {
	if Serst(false, false, false) != 0xC0 || Serst(false, true, false) != 0xC8 || Serst(true, true, false) != 0xCC {
		t.Fatal("serst")
	}
	if tp, _ := Tempo(2, AlwaysOn); tp != 0xE2 {
		t.Fatalf("tempo %02X", tp)
	}
	if _, err := Tempo(0, AlwaysOn); err == nil {
		t.Fatal("tempo 0 accepted")
	}
	if Function(Appear, false, false) != 0xC1 || Function(Slide, false, false) != 0xCF {
		t.Fatal("function")
	}
	if PageStatus(0, false, false, false) != 0x80 || PageStatus(0b11, false, false, false) != 0x83 || PageStatus(0, false, false, true) != 0xC0 {
		t.Fatal("status")
	}
}

func TestClockPage(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 5, 7, 0, time.Local) // a Monday
	f, _ := BuildFrame(ClockPage(now), 4, 0, 0xC8)
	if string(f[8:20]) != "090507121002" {
		t.Fatalf("clock text %q", f[8:20])
	}
}

func TestScheduleBitMustMatch(t *testing.T) {
	if _, err := BuildFrame(NewPage(1, []byte("x")), 4, 0, 0xD0); err == nil {
		t.Fatal("schedule bit without block accepted")
	}
}
