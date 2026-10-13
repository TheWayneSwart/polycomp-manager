//go:build windows

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"polycomp/winapp/internal/gtxc"
	"polycomp/winapp/internal/link"
)

const appTitle = "Polycomp Manager"

type rowEditor struct {
	group  *walk.GroupBox
	text   *walk.LineEdit
	count  *walk.Label
	colour *walk.ComboBox
	flash  *walk.CheckBox
	wide   *walk.CheckBox
	align  *walk.ComboBox
}

type app struct {
	settings Settings
	msg      Message
	cur      int // selected page index
	loading  bool

	mw       *walk.MainWindow
	portCB   *walk.ComboBox
	addrNE   *walk.NumberEdit
	status   *walk.Label
	pagesLB  *walk.ListBox
	layoutCB *walk.ComboBox
	effectCB *walk.ComboBox
	tempoCB  *walk.ComboBox
	rows     [4]rowEditor
	prev     preview
	logTE    *walk.TextEdit
	logGroup *walk.GroupBox
	sendBtn  *walk.PushButton
	busy     bool
}

func run() error {
	a := &app{settings: LoadSettings()}
	a.msg = a.settings.Last
	if len(a.msg.Pages) == 0 {
		a.msg.Pages = []PageModel{NewPageModel()}
	}
	var err error
	a.prev.smallFont, err = walk.NewFont("Consolas", 17, walk.FontBold)
	if err != nil {
		return err
	}
	a.prev.tallFont, _ = walk.NewFont("Consolas", 36, walk.FontBold)
	a.prev.page = a.currentPageFromEditor

	rowGroups := make([]Widget, 4)
	for i := range rowGroups {
		rowGroups[i] = a.rowEditorWidget(i)
	}

	win := MainWindow{
		AssignTo: &a.mw,
		Title:    appTitle,
		MinSize:  Size{Width: 980, Height: 640},
		Size:     Size{Width: 1040, Height: 720},
		Layout:   VBox{},
		MenuItems: []MenuItem{
			Menu{Text: "&File", Items: []MenuItem{
				Action{Text: "&Open message...", OnTriggered: a.openMessage},
				Action{Text: "&Save message as...", OnTriggered: a.saveMessage},
				Separator{},
				Action{Text: "E&xit", OnTriggered: func() { a.mw.Close() }},
			}},
			Menu{Text: "&Sign", Items: []MenuItem{
				Action{Text: "&Send to sign", OnTriggered: a.send},
				Action{Text: "Set sign &clock to this PC's time", OnTriggered: a.setClock},
				Action{Text: "Show &frame bytes", OnTriggered: a.showHex},
			}},
			Menu{Text: "&Help", Items: []MenuItem{
				Action{Text: "&About", OnTriggered: a.about},
			}},
		},
		Children: []Widget{
			// connection bar
			GroupBox{Title: "Connection", Layout: HBox{}, Children: []Widget{
				Label{Text: "Serial port:"},
				ComboBox{AssignTo: &a.portCB, MinSize: Size{Width: 120}, Editable: true},
				PushButton{Text: "Refresh", OnClicked: a.refreshPorts},
				Label{Text: "Sign address (0 = all signs):"},
				NumberEdit{AssignTo: &a.addrNE, MinValue: 0, MaxValue: 127, Decimals: 0, MinSize: Size{Width: 60}},
				HSpacer{},
				Label{AssignTo: &a.status, Text: "Ready", MinSize: Size{Width: 260}},
			}},
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
				// pages list
				GroupBox{Title: "Pages (shown one after another)", Layout: VBox{}, MaxSize: Size{Width: 220}, Children: []Widget{
					ListBox{AssignTo: &a.pagesLB, MinSize: Size{Height: 120}, OnCurrentIndexChanged: a.pageSelected},
					Composite{Layout: Grid{Columns: 2, MarginsZero: true}, Children: []Widget{
						PushButton{Text: "Add page", OnClicked: a.addPage},
						PushButton{Text: "Remove", OnClicked: a.removePage},
						PushButton{Text: "Move up", OnClicked: func() { a.movePage(-1) }},
						PushButton{Text: "Move down", OnClicked: func() { a.movePage(1) }},
					}},
					VSpacer{},
				}},
				// page editor
				GroupBox{Title: "Page content", Layout: VBox{}, Children: append([]Widget{
					Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
						Label{Text: "Layout:"},
						ComboBox{AssignTo: &a.layoutCB, Model: gtxc.LayoutNames(), OnCurrentIndexChanged: a.layoutChanged, MinSize: Size{Width: 220}},
						Label{Text: "Effect:"},
						ComboBox{AssignTo: &a.effectCB, Model: gtxc.EffectNames(), OnCurrentIndexChanged: a.effectChanged, MinSize: Size{Width: 200}},
						Label{Text: "Time / speed:"},
						ComboBox{AssignTo: &a.tempoCB, Model: tempoLabels(false), OnCurrentIndexChanged: a.editorChanged, MinSize: Size{Width: 140}},
						HSpacer{},
					}},
				}, rowGroups...)},
			}},
			GroupBox{Title: "Preview (what the board will show)", Layout: HBox{}, Children: []Widget{
				HSpacer{},
				CustomWidget{AssignTo: &a.prev.widget, MinSize: Size{Width: 192 * scale, Height: 32 * scale},
					MaxSize: Size{Width: 192 * scale, Height: 32 * scale}, ClearsBackground: true, InvalidatesOnResize: true,
					Paint: a.prev.paint},
				HSpacer{},
			}},
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
				PushButton{AssignTo: &a.sendBtn, Text: "   Send to sign   ", OnClicked: a.send, MinSize: Size{Height: 36}},
				PushButton{Text: "Set sign clock", OnClicked: a.setClock},
				PushButton{Text: "Open message...", OnClicked: a.openMessage},
				PushButton{Text: "Save message...", OnClicked: a.saveMessage},
				HSpacer{},
				CheckBox{Text: "Show log", Checked: a.settings.ShowLog, OnCheckedChanged: a.toggleLog, AssignTo: new(*walk.CheckBox)},
			}},
			GroupBox{AssignTo: &a.logGroup, Title: "Log", Layout: VBox{}, Visible: a.settings.ShowLog, Children: []Widget{
				TextEdit{AssignTo: &a.logTE, ReadOnly: true, VScroll: true, MinSize: Size{Height: 100}},
			}},
		},
	}
	if err := win.Create(); err != nil {
		return err
	}
	a.refreshPorts()
	if a.settings.Port != "" {
		a.portCB.SetText(a.settings.Port)
	}
	a.addrNE.SetValue(float64(a.settings.Address))
	a.reloadPagesList(0)
	a.mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) { a.persist() })
	a.mw.Run()
	return nil
}

func (a *app) rowEditorWidget(i int) Widget {
	re := &a.rows[i]
	return GroupBox{AssignTo: &re.group, Title: fmt.Sprintf("Row %d", i+1), Layout: HBox{}, Children: []Widget{
		LineEdit{AssignTo: &re.text, MinSize: Size{Width: 360}, MaxLength: 32, OnTextChanged: a.editorChanged},
		Label{AssignTo: &re.count, Text: "0/32", MinSize: Size{Width: 44}},
		Label{Text: "Colour:"},
		ComboBox{AssignTo: &re.colour, Model: gtxc.ColourNames(), OnCurrentIndexChanged: a.editorChanged, MinSize: Size{Width: 80}},
		CheckBox{AssignTo: &re.flash, Text: "Flash", OnCheckedChanged: a.editorChanged},
		CheckBox{AssignTo: &re.wide, Text: "Wide", OnCheckedChanged: a.editorChanged},
		Label{Text: "Align:"},
		ComboBox{AssignTo: &re.align, Model: gtxc.AlignNames(), OnCurrentIndexChanged: a.editorChanged, MinSize: Size{Width: 80}},
	}}
}

func tempoLabels(slide bool) []string {
	out := make([]string, 9)
	for i := 1; i <= 9; i++ {
		if slide {
			out[i-1] = fmt.Sprintf("Speed %d", i)
			if i == 1 {
				out[i-1] += " (fastest)"
			} else if i == 9 {
				out[i-1] += " (slowest)"
			}
		} else {
			out[i-1] = fmt.Sprintf("%d seconds", gtxc.TempoSeconds[i])
		}
	}
	return out
}

// ---- editor <-> model ----

func (a *app) currentPageFromEditor() PageModel {
	p := a.msg.Pages[a.cur]
	if a.layoutCB == nil || a.layoutCB.CurrentIndex() < 0 {
		return p
	}
	p.Layout = a.layoutCB.CurrentIndex()
	p.Effect = max(a.effectCB.CurrentIndex(), 0)
	p.Tempo = max(a.tempoCB.CurrentIndex(), 0) + 1
	for i := range a.rows {
		re := &a.rows[i]
		p.Rows[i] = RowModel{Text: re.text.Text(), Colour: max(re.colour.CurrentIndex(), 0), Flash: re.flash.Checked(),
			Wide: re.wide.Checked(), Align: max(re.align.CurrentIndex(), 0)}
	}
	return p
}

func (a *app) loadEditor(p PageModel) {
	a.loading = true
	defer func() { a.loading = false }()
	a.layoutCB.SetCurrentIndex(p.Layout)
	a.effectCB.SetCurrentIndex(p.Effect)
	a.tempoCB.SetModel(tempoLabels(gtxc.Effect(p.Effect) == gtxc.Slide))
	a.tempoCB.SetCurrentIndex(p.Tempo - 1)
	for i := range a.rows {
		re := &a.rows[i]
		re.text.SetText(p.Rows[i].Text)
		re.colour.SetCurrentIndex(p.Rows[i].Colour)
		re.flash.SetChecked(p.Rows[i].Flash)
		re.wide.SetChecked(p.Rows[i].Wide)
		re.align.SetCurrentIndex(p.Rows[i].Align)
	}
	a.applyLayoutToRows(gtxc.LayoutMode(p.Layout))
	a.prev.widget.Invalidate()
}

func (a *app) applyLayoutToRows(mode gtxc.LayoutMode) {
	geo := mode.Rows()
	for i := range a.rows {
		re := &a.rows[i]
		if i < len(geo) {
			tall := geo[i][0] == gtxc.TallVisible
			name := fmt.Sprintf("Row %d", i+1)
			if tall {
				name = fmt.Sprintf("Big row %d", i+1)
			}
			re.group.SetTitle(name)
			re.group.SetVisible(true)
			re.text.SetMaxLength(geo[i][0])
		} else {
			re.group.SetVisible(false)
		}
	}
	a.updateCounts()
}

func (a *app) updateCounts() {
	geo := gtxc.LayoutMode(max(a.layoutCB.CurrentIndex(), 0)).Rows()
	for i := range a.rows {
		if i >= len(geo) {
			continue
		}
		re := &a.rows[i]
		n := len([]rune(re.text.Text()))
		if re.wide.Checked() {
			n *= 2
		}
		re.count.SetText(fmt.Sprintf("%d/%d", n, geo[i][0]))
	}
}

func (a *app) editorChanged() {
	if a.loading {
		return
	}
	a.msg.Pages[a.cur] = a.currentPageFromEditor()
	a.updateCounts()
	a.prev.widget.Invalidate()
	a.refreshPageTitles()
}

func (a *app) layoutChanged() {
	if a.loading {
		return
	}
	a.applyLayoutToRows(gtxc.LayoutMode(max(a.layoutCB.CurrentIndex(), 0)))
	a.editorChanged()
}

func (a *app) effectChanged() {
	if a.loading {
		return
	}
	slide := gtxc.Effect(a.effectCB.CurrentIndex()) == gtxc.Slide
	idx := a.tempoCB.CurrentIndex()
	a.loading = true
	a.tempoCB.SetModel(tempoLabels(slide))
	a.tempoCB.SetCurrentIndex(max(idx, 0))
	a.loading = false
	a.editorChanged()
}

// ---- pages ----

func pageTitle(i int, p PageModel) string {
	t := strings.TrimSpace(p.Rows[0].Text)
	if t == "" {
		t = "(empty)"
	}
	if len(t) > 18 {
		t = t[:18] + "…"
	}
	return fmt.Sprintf("%d: %s", i+1, t)
}

func (a *app) refreshPageTitles() {
	titles := make([]string, len(a.msg.Pages))
	for i, p := range a.msg.Pages {
		titles[i] = pageTitle(i, p)
	}
	cur := a.pagesLB.CurrentIndex()
	a.loading = true
	a.pagesLB.SetModel(titles)
	a.pagesLB.SetCurrentIndex(cur)
	a.loading = false
}

func (a *app) reloadPagesList(sel int) {
	a.cur = sel
	a.refreshPageTitles()
	a.pagesLB.SetCurrentIndex(sel)
	a.loadEditor(a.msg.Pages[sel])
}

func (a *app) pageSelected() {
	if a.loading {
		return
	}
	i := a.pagesLB.CurrentIndex()
	if i < 0 || i >= len(a.msg.Pages) || i == a.cur {
		return
	}
	a.msg.Pages[a.cur] = a.currentPageFromEditor()
	a.cur = i
	a.loadEditor(a.msg.Pages[i])
}

func (a *app) addPage() {
	a.msg.Pages[a.cur] = a.currentPageFromEditor()
	p := NewPageModel()
	p.Layout = a.msg.Pages[a.cur].Layout
	a.msg.Pages = append(a.msg.Pages, p)
	a.reloadPagesList(len(a.msg.Pages) - 1)
}

func (a *app) removePage() {
	if len(a.msg.Pages) == 1 {
		a.msg.Pages[0] = NewPageModel()
		a.reloadPagesList(0)
		return
	}
	a.msg.Pages = append(a.msg.Pages[:a.cur], a.msg.Pages[a.cur+1:]...)
	a.reloadPagesList(min(a.cur, len(a.msg.Pages)-1))
}

func (a *app) movePage(d int) {
	j := a.cur + d
	if j < 0 || j >= len(a.msg.Pages) {
		return
	}
	a.msg.Pages[a.cur] = a.currentPageFromEditor()
	a.msg.Pages[a.cur], a.msg.Pages[j] = a.msg.Pages[j], a.msg.Pages[a.cur]
	a.reloadPagesList(j)
}

// ---- files & settings ----

func (a *app) openMessage() {
	dlg := walk.FileDialog{Title: "Open message", Filter: "Sign messages (*.json)|*.json|All files (*.*)|*.*"}
	if ok, _ := dlg.ShowOpen(a.mw); !ok {
		return
	}
	m, err := LoadMessageFile(dlg.FilePath)
	if err != nil {
		walk.MsgBox(a.mw, appTitle, "Could not open that file:\n"+err.Error(), walk.MsgBoxIconError)
		return
	}
	a.msg = m
	a.reloadPagesList(0)
}

func (a *app) saveMessage() {
	a.msg.Pages[a.cur] = a.currentPageFromEditor()
	dlg := walk.FileDialog{Title: "Save message", Filter: "Sign messages (*.json)|*.json", FilePath: "message.json"}
	if ok, _ := dlg.ShowSave(a.mw); !ok {
		return
	}
	path := dlg.FilePath
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}
	if err := a.msg.SaveFile(path); err != nil {
		walk.MsgBox(a.mw, appTitle, "Could not save:\n"+err.Error(), walk.MsgBoxIconError)
	}
}

func (a *app) persist() {
	a.msg.Pages[a.cur] = a.currentPageFromEditor()
	a.settings.Port = a.portCB.Text()
	a.settings.Address = int(a.addrNE.Value())
	a.settings.Last = a.msg
	_ = a.settings.Save()
}

func (a *app) toggleLog() {
	a.settings.ShowLog = !a.settings.ShowLog
	a.logGroup.SetVisible(a.settings.ShowLog)
}

func (a *app) logf(format string, args ...any) {
	line := time.Now().Format("15:04:05") + " " + fmt.Sprintf(format, args...)
	a.mw.Synchronize(func() { a.logTE.AppendText(line + "\r\n") })
}

func (a *app) setStatus(s string) { a.mw.Synchronize(func() { a.status.SetText(s) }) }

// ---- sign actions ----

func (a *app) refreshPorts() {
	ports, err := link.Ports()
	if err != nil {
		a.logf("port list: %v", err)
	}
	cur := a.portCB.Text()
	a.portCB.SetModel(ports)
	if cur != "" {
		a.portCB.SetText(cur)
	} else if len(ports) > 0 {
		a.portCB.SetCurrentIndex(len(ports) - 1) // USB adapters are usually the highest COM number
	}
}

func (a *app) frames() ([][]byte, bool) {
	a.msg.Pages[a.cur] = a.currentPageFromEditor()
	frames, err := a.msg.Frames(a.settings.Nol, int(a.addrNE.Value()))
	if err != nil {
		walk.MsgBox(a.mw, appTitle, err.Error(), walk.MsgBoxIconWarning)
		return nil, false
	}
	return frames, true
}

func (a *app) showHex() {
	frames, ok := a.frames()
	if !ok {
		return
	}
	if !a.settings.ShowLog {
		a.toggleLog()
	}
	for i, f := range frames {
		a.logf("page %d (%d bytes): %s", i+1, len(f), gtxc.Hex(f))
	}
}

func (a *app) transmit(what string, frames [][]byte) {
	if a.busy {
		return
	}
	port := strings.TrimSpace(a.portCB.Text())
	if port == "" {
		walk.MsgBox(a.mw, appTitle, "Choose the serial port the sign is plugged into first.", walk.MsgBoxIconWarning)
		return
	}
	a.busy = true
	a.sendBtn.SetEnabled(false)
	a.setStatus("Sending " + what + "...")
	go func() {
		err := func() error {
			l, err := link.Open(port, func(s string) { a.logf("%s", s) })
			if err != nil {
				return err
			}
			defer l.Close()
			return l.SendPages(frames)
		}()
		a.mw.Synchronize(func() {
			a.busy = false
			a.sendBtn.SetEnabled(true)
			if err != nil {
				a.status.SetText("Failed: " + err.Error())
				walk.MsgBox(a.mw, appTitle, "The sign did not accept the "+what+":\n\n"+err.Error()+
					"\n\nCheck the cable, that the right port is selected, and that the sign is on.", walk.MsgBoxIconError)
				return
			}
			a.status.SetText("Sent " + what + " at " + time.Now().Format("15:04:05") + ", sign acknowledged")
			a.persist()
		})
	}()
}

func (a *app) send() {
	frames, ok := a.frames()
	if !ok {
		return
	}
	a.transmit(fmt.Sprintf("%d page(s)", len(frames)), frames)
}

func (a *app) setClock() {
	f, err := ClockFrame(a.settings.Nol, int(a.addrNE.Value()))
	if err != nil {
		walk.MsgBox(a.mw, appTitle, err.Error(), walk.MsgBoxIconError)
		return
	}
	a.transmit("clock", [][]byte{f})
}

func (a *app) about() {
	walk.MsgBox(a.mw, "About "+appTitle,
		appTitle+"\n\nSends text pages to PolyComp AMP2008 LED signs over a serial cable using the GTXC protocol.\n"+
			"Rows hold 32 characters (16 when big). Wide characters count double.\n"+
			"Messages are saved as .json files and the last message is restored on start.",
		walk.MsgBoxIconInformation)
}
