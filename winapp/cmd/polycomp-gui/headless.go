package main

import (
	"flag"
	"fmt"
	"os"

	"polycomp/winapp/internal/gtxc"
	"polycomp/winapp/internal/link"
)

// runHeadless lets the same executable be scripted:
//
//	polycomp-gui.exe --send message.json --port COM3 [--address 0] [--dry-run]
//	polycomp-gui.exe --set-clock --port COM3
func runHeadless(args []string) error {
	fs := flag.NewFlagSet("polycomp-gui", flag.ContinueOnError)
	file := fs.String("send", "", "message .json file to send")
	clock := fs.Bool("set-clock", false, "set the sign clock to this PC's time")
	port := fs.String("port", "", "serial port, e.g. COM3")
	addr := fs.Int("address", 0, "sign address, 0 = all signs")
	nol := fs.Int("nol", 0, "Nol byte: 0 = any sign, else the sign's line count")
	dry := fs.Bool("dry-run", false, "print frames as hex, do not open the port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var frames [][]byte
	if *file != "" {
		m, err := LoadMessageFile(*file)
		if err != nil {
			return err
		}
		frames, err = m.Frames(*nol, *addr)
		if err != nil {
			return err
		}
	} else if *clock {
		f, err := ClockFrame(*nol, *addr)
		if err != nil {
			return err
		}
		frames = [][]byte{f}
	} else {
		fs.Usage()
		return fmt.Errorf("nothing to do")
	}
	if *dry {
		for _, f := range frames {
			fmt.Println(gtxc.Hex(f))
		}
		return nil
	}
	if *port == "" {
		return fmt.Errorf("--port is required")
	}
	l, err := link.Open(*port, func(s string) { fmt.Fprintln(os.Stderr, s) })
	if err != nil {
		return err
	}
	defer l.Close()
	return l.SendPages(frames)
}
