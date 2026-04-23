//go:build !windows

package main

import "log"

func runWithTray(startFn func() error) {
	if err := startFn(); err != nil {
		log.Fatal(err)
	}
}
