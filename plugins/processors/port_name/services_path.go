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
		log.Fatal("E! WINDIR Environment var is unset"
	}
	return filepath.Join(windir, `system32\drivers\etc\services`)
}
