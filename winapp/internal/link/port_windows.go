//go:build windows

package link

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Plain synchronous Win32 serial I/O (no overlapped reads): works on Windows and under Wine.

type port struct {
	h windows.Handle
}

type dcb struct {
	DCBlength, BaudRate uint32
	Flags               uint32
	wReserved           uint16
	XonLim, XoffLim     uint16
	ByteSize, Parity    byte
	StopBits            byte
	XonChar, XoffChar   byte
	ErrorChar, EofChar  byte
	EvtChar             byte
	wReserved1          uint16
}

type commTimeouts struct {
	ReadIntervalTimeout, ReadTotalTimeoutMultiplier, ReadTotalTimeoutConstant uint32
	WriteTotalTimeoutMultiplier, WriteTotalTimeoutConstant                    uint32
}

var (
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	procGetCommState   = kernel32.NewProc("GetCommState")
	procSetCommState   = kernel32.NewProc("SetCommState")
	procSetCommTimeout = kernel32.NewProc("SetCommTimeouts")
	procPurgeComm      = kernel32.NewProc("PurgeComm")
)

func openPort(name string) (*port, error) {
	path := name
	if !strings.HasPrefix(path, `\\.\`) {
		path = `\\.\` + name
	}
	p16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p16, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	var d dcb
	d.DCBlength = uint32(unsafe.Sizeof(d))
	if r, _, e := procGetCommState.Call(uintptr(h), uintptr(unsafe.Pointer(&d))); r == 0 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("GetCommState: %v", e)
	}
	d.BaudRate = 9600
	d.ByteSize = 8
	d.Parity = 0   // none
	d.StopBits = 0 // one
	d.Flags = 1    // fBinary only: no parity check, no DTR/RTS control, no flow control
	if r, _, e := procSetCommState.Call(uintptr(h), uintptr(unsafe.Pointer(&d))); r == 0 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("SetCommState: %v", e)
	}
	p := &port{h: h}
	_ = p.setReadTimeout(100 * time.Millisecond)
	const purgeRX = 0x0008 | 0x0002 // PURGE_RXCLEAR | PURGE_RXABORT
	procPurgeComm.Call(uintptr(h), purgeRX)
	return p, nil
}

func (p *port) setReadTimeout(d time.Duration) error {
	t := commTimeouts{ReadIntervalTimeout: 0xFFFFFFFF, ReadTotalTimeoutMultiplier: 0xFFFFFFFF,
		ReadTotalTimeoutConstant: uint32(d / time.Millisecond), WriteTotalTimeoutConstant: 2000}
	if r, _, e := procSetCommTimeout.Call(uintptr(p.h), uintptr(unsafe.Pointer(&t))); r == 0 {
		return fmt.Errorf("SetCommTimeouts: %v", e)
	}
	return nil
}

func (p *port) Write(b []byte) (int, error) {
	var n uint32
	err := windows.WriteFile(p.h, b, &n, nil)
	return int(n), err
}

// Read returns after up to the read timeout with whatever arrived (possibly nothing).
func (p *port) Read(b []byte) (int, error) {
	var n uint32
	err := windows.ReadFile(p.h, b, &n, nil)
	return int(n), err
}

func (p *port) Close() error { return windows.CloseHandle(p.h) }

// listPorts reads HKLM\HARDWARE\DEVICEMAP\SERIALCOMM, which Windows and Wine both populate.
func listPorts() ([]string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DEVICEMAP\SERIALCOMM`, registry.READ)
	if err != nil {
		return nil, nil // no serial ports at all
	}
	defer k.Close()
	names, err := k.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, n := range names {
		v, _, err := k.GetStringValue(n)
		if err == nil && v != "" {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})
	return out, nil
}
