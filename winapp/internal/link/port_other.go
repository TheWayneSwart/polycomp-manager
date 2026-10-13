//go:build !windows

package link

import (
	"sort"
	"time"

	"go.bug.st/serial"
)

type port struct{ p serial.Port }

func openPort(name string) (*port, error) {
	p, err := serial.Open(name, &serial.Mode{BaudRate: 9600, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit})
	if err != nil {
		return nil, err
	}
	_ = p.ResetInputBuffer()
	_ = p.SetReadTimeout(100 * time.Millisecond)
	return &port{p: p}, nil
}

func (p *port) setReadTimeout(d time.Duration) error { return p.p.SetReadTimeout(d) }
func (p *port) Write(b []byte) (int, error)          { return p.p.Write(b) }
func (p *port) Read(b []byte) (int, error)           { return p.p.Read(b) }
func (p *port) Close() error                         { return p.p.Close() }

func listPorts() ([]string, error) {
	ps, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}
	sort.Strings(ps)
	return ps, nil
}
