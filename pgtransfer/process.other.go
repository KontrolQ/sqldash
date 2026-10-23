//go:build !linux

package pgtransfer

import (
	"os"
	"os/exec"
)

func executionIdentity() error {
	return UnsupportedPlatform
}

func protectDirectory(string) error {
	return UnsupportedPlatform
}

func protectFile(*os.File) error {
	return UnsupportedPlatform
}

func isolateCommand(*exec.Cmd) error {
	return UnsupportedPlatform
}

func terminateCommand(*exec.Cmd) {}
