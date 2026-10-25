//go:build !linux

package redishost

import (
	"net"
	"os"
	"os/exec"
)

func killOwnedProcess(uint, int, string) error { return Unavailable }

func socketPeerMatches(net.Conn, uint, int) bool { return false }

func managedSupported() bool                           { return false }
func instanceCredentials(uint) (int, int, error)       { return 0, 0, Unavailable }
func verifyOwner(os.FileInfo, int) bool                { return false }
func lockOwnedFile(string, bool) (*os.File, error)     { return nil, Unavailable }
func configureProcess(*exec.Cmd, uint, *os.File) error { return Unavailable }
func processStartIdentity(int) string                  { return "" }
