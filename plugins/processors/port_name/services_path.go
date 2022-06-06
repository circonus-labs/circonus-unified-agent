//go:build windows
// +build windows

package portname

import (
	"log"
	"os"
	"path/filepath"
)

func servicesPath() string {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		log.Print("I! WINDIR Environment var is unset")
		// use a sensible default
		windir = "C:\\Windows"
	}
	return filepath.Join(windir, `system32\drivers\etc\services`)
}
