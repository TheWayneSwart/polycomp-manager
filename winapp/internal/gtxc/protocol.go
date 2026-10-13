// Package gtxc encodes PolyComp GTXC frames for the AMP2008 sign controller.
//
// Facts verified live on 2026-10-12 (see ../../CLAUDE.md): never set SERST B1 (interrupt mode),
// the firmware drops such frames; Nol must be 0 or the configured line count (4); PAGE STATUS B6
// floods the board green, so the default status is 0x80.
package gtxc

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

const (
	SYNC = 0x00
	ETX  = 0x03
	EOT  = 0x04
	ACK  = 0x06
	NAK  = 0x15
	CTRL = 0x1C // inline formatting prefix
)

// Effect is the FUNCTION byte's low nibble.
type Effect int

const (
	Auto Effect = iota
	Appear
	Wipe
	Open
	Lock
	Rotate
	Right
	Left
	RollUp
	RollDown
	PingPong
	FillUp
	Paint
	FadeIn
	Jump
	Slide // keeps upper rows static, scrolls row 4 right-to-left; TEMPO is then the speed
)

var effectNames = []string{"Auto (random)", "Appear", "Wipe", "Open", "Lock", "Rotate", "Right", "Left",
	"Roll up", "Roll down", "Ping pong", "Fill up", "Paint", "Fade in", "Jump", "Scroll bottom row (slide)"}

func (e Effect) String() string { return effectNames[e] }

// EffectNames lists the effects in FUNCTION order, for UI pickers.
func EffectNames() []string { return append([]string(nil), effectNames...) }

// TempoSeconds maps TEMPO index 1..9 to seconds on screen (in Slide mode it is the speed, 1 fastest).
var TempoSeconds = [10]int{0, 2, 5, 10, 20, 30, 45, 60, 90, 120}

type DisplayMode int

const (
	Timer     DisplayMode = 0 // follow schedule
	AlwaysOn  DisplayMode = 2 // B5 (vendor example 0xE2)
	AlwaysOff DisplayMode = 1 // B4
)

// Serst builds the serial status byte. Interrupt mode is deliberately not offered.
func Serst(morePages, ack, schedule bool) byte {
	v := byte(0xC0)
	if morePages {
		v |= 1 << 2
	}
	if ack {
		v |= 1 << 3
	}
	if schedule {
		v |= 1 << 4
	}
	return v
}

func Tempo(index int, mode DisplayMode) (byte, error) {
	if index < 1 || index > 9 {
		return 0, fmt.Errorf("tempo index must be 1-9, got %d", index)
	}
	return 0xC0 | byte(mode)<<4 | byte(index), nil
}

func Function(e Effect, clock, temperature bool) byte {
	v := 0xC0 | byte(e)&0x0F
	if clock {
		v |= 1 << 4
	}
	if temperature {
		v |= 1 << 5
	}
	return v
}

// PageStatus: boldPairs bit0 joins rows 1+2, bit1 rows 3+4 (16 px tall rows).
func PageStatus(boldPairs byte, autoCentre, foreign, background bool) byte {
	v := 0x80 | boldPairs&0x0F
	if autoCentre {
		v |= 1 << 4
	}
	if foreign {
		v |= 1 << 5
	}
	if background {
		v |= 1 << 6
	}
	return v
}

// Fmt returns an inline control block: F flash, E enlarge (double width), R/G/Y/M/W colour, D default.
func Fmt(c byte) []byte { return []byte{CTRL, c} }

type Page struct {
	Number   int
	Tempo    byte
	Function byte
	Status   byte
	Text     []byte // already CP437-encoded, may contain 0x1C blocks
	Schedule []byte // 17 bytes when the schedule bit is set, else nil
}

// NewPage is a convenience with the proven defaults: 5 s, Appear, status 0x80.
func NewPage(number int, text []byte) Page {
	t, _ := Tempo(2, AlwaysOn)
	return Page{Number: number, Tempo: t, Function: Function(Appear, false, false), Status: PageStatus(0, false, false, false), Text: text}
}

func Checksum(data []byte) byte {
	var c byte
	for _, b := range data {
		c ^= b
	}
	return c
}

// BuildFrame encodes one page. nol is 0 or the sign's configured line count; address 0 = all-call.
func BuildFrame(p Page, nol, address int, flags byte) ([]byte, error) {
	if address < 0 || address > 127 {
		return nil, fmt.Errorf("address must be 0-127")
	}
	if p.Number < 0 || p.Number > 999 {
		return nil, fmt.Errorf("page number must be 0-999")
	}
	if (flags&0x10 != 0) != (len(p.Schedule) > 0) {
		return nil, fmt.Errorf("schedule bit must match presence of schedule block")
	}
	if len(p.Schedule) > 0 && len(p.Schedule) != 17 {
		return nil, fmt.Errorf("schedule block must be 17 bytes")
	}
	f := []byte{SYNC, byte(nol), byte(address), ETX, flags}
	f = append(f, []byte(fmt.Sprintf("%03d", p.Number))...)
	if p.Number == 0 {
		f = append(f, p.Text...) // clock page: no tempo/function/status
	} else {
		f = append(f, p.Schedule...)
		f = append(f, p.Tempo, p.Function, p.Status)
		f = append(f, p.Text...)
	}
	f = append(f, EOT)
	return append(f, Checksum(f)), nil
}

// ClockPage is page 000: HHMMSSDDMM0W with W = day of week, 1 = Sunday.
func ClockPage(now time.Time) Page {
	dow := int(now.Weekday()) + 1
	return Page{Number: 0, Text: []byte(now.Format("150405") + now.Format("0201") + fmt.Sprintf("0%d", dow))}
}

// EncodeCP437 converts text to the sign's "extended IBM ASCII".
func EncodeCP437(s string) ([]byte, error) {
	return charmap.CodePage437.NewEncoder().Bytes([]byte(s))
}

func Hex(b []byte) string {
	var sb strings.Builder
	for i, x := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%02X", x)
	}
	return sb.String()
}
