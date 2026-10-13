//go:build windows

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		if err := runHeadless(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
