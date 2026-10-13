package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"polycomp/winapp/internal/gtxc"
)

// RowModel is what the user edits for one visual row.
type RowModel struct {
	Text   string `json:"text"`
	Colour int    `json:"colour"` // gtxc.Colour
	Flash  bool   `json:"flash"`
	Wide   bool   `json:"wide"`
	Align  int    `json:"align"` // gtxc.Align
}

// PageModel is one page of a message sequence.
type PageModel struct {
	Layout int         `json:"layout"` // gtxc.LayoutMode
	Rows   [4]RowModel `json:"rows"`
	Effect int         `json:"effect"` // gtxc.Effect
	Tempo  int         `json:"tempo"`  // 1-9
}

func NewPageModel() PageModel {
	return PageModel{Layout: int(gtxc.TallTop), Effect: int(gtxc.Appear), Tempo: 2}
}

// Message is a whole sequence, saved as a .json file.
type Message struct {
	Pages []PageModel `json:"pages"`
}

// Settings persist between runs.
type Settings struct {
	Port    string  `json:"port"`
	Address int     `json:"address"`
	Nol     int     `json:"nol"` // 0 = any sign, else must equal the sign's configured line count
	ShowLog bool    `json:"show_log"`
	Last    Message `json:"last_message"`
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "PolycompManager")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func LoadSettings() Settings {
	s := Settings{Nol: 0, Last: Message{Pages: []PageModel{NewPageModel()}}}
	p, err := settingsPath()
	if err != nil {
		return s
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	if len(s.Last.Pages) == 0 {
		s.Last.Pages = []PageModel{NewPageModel()}
	}
	return s
}

func (s Settings) Save() error {
	p, err := settingsPath()
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(p, b, 0o644)
}

func (m Message) SaveFile(path string) error {
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(path, b, 0o644)
}

func LoadMessageFile(path string) (Message, error) {
	var m Message
	b, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	if len(m.Pages) == 0 {
		return m, fmt.Errorf("file contains no pages")
	}
	return m, nil
}

// rows returns the rows that exist for the page's layout.
func (p PageModel) rows() []gtxc.Row {
	n := len(gtxc.LayoutMode(p.Layout).Rows())
	out := make([]gtxc.Row, n)
	for i := 0; i < n; i++ {
		r := p.Rows[i]
		out[i] = gtxc.Row{Text: r.Text, Colour: gtxc.Colour(r.Colour), Flash: r.Flash, Wide: r.Wide, Align: gtxc.Align(r.Align)}
	}
	// drop trailing empty rows so the last row is not padded needlessly
	for len(out) > 1 && out[len(out)-1].Text == "" {
		out = out[:len(out)-1]
	}
	return out
}

// Frames encodes the whole message as a page sequence.
func (m Message) Frames(nol, address int) ([][]byte, error) {
	var frames [][]byte
	for i, p := range m.Pages {
		text, err := gtxc.Layout(p.rows(), gtxc.LayoutMode(p.Layout))
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", i+1, err)
		}
		tempo, err := gtxc.Tempo(p.Tempo, gtxc.AlwaysOn)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", i+1, err)
		}
		page := gtxc.Page{
			Number:   i + 1,
			Tempo:    tempo,
			Function: gtxc.Function(gtxc.Effect(p.Effect), false, false),
			Status:   gtxc.PageStatus(gtxc.LayoutMode(p.Layout).BoldBits(), false, false, false),
			Text:     text,
		}
		more := i < len(m.Pages)-1
		f, err := gtxc.BuildFrame(page, nol, address, gtxc.Serst(more, true, false))
		if err != nil {
			return nil, err
		}
		frames = append(frames, f)
	}
	return frames, nil
}

func ClockFrame(nol, address int) ([]byte, error) {
	return gtxc.BuildFrame(gtxc.ClockPage(time.Now()), nol, address, gtxc.Serst(false, true, false))
}
