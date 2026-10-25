package redishost

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func socketPeerMatches(connection net.Conn, databaseID uint, processID int) bool {
	transport, valid := connection.(*net.UnixConn)
	if !valid || processID < 1 {
		return false
	}
	userID, _, err := instanceCredentials(databaseID)
	if err != nil {
		return false
	}
	control, err := transport.SyscallConn()
	if err != nil {
		return false
	}
	matched := false
	if err := control.Control(func(descriptor uintptr) {
		credentials, err := unix.GetsockoptUcred(int(descriptor), unix.SOL_SOCKET, unix.SO_PEERCRED)
		matched = err == nil && credentials.Uid == uint32(userID) && credentials.Pid == int32(processID)
	}); err != nil {
		return false
	}
	return matched
}

func killOwnedProcess(databaseID uint, processID int, start string) error {
	if processID < 1 || start == "" || processStartIdentity(processID) != start {
		return nil
	}
	userID, _, err := instanceCredentials(databaseID)
	if err != nil {
		return err
	}
	information, err := os.Stat("/proc/" + strconv.Itoa(processID))
	if err != nil || !verifyOwner(information, userID) {
		return IdentityMismatch
	}
	process, err := os.FindProcess(processID)
	if err != nil {
		return PolicyUnconfirmed
	}
	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return PolicyUnconfirmed
	}
	return nil
}

func managedSupported() bool {
	return os.Geteuid() == 0
}

func instanceCredentials(databaseID uint) (int, int, error) {
	if databaseID == 0 || databaseID > maximumDatabaseID {
		return 0, 0, IdentityMismatch
	}
	identifier := instanceUserBase + int(databaseID)
	return identifier, identifier, nil
}

func verifyOwner(information os.FileInfo, userID int) bool {
	attributes, valid := information.Sys().(*syscall.Stat_t)
	return valid && attributes.Uid == uint32(userID) && attributes.Nlink == 1 || valid && information.IsDir() && attributes.Uid == uint32(userID)
}

func lockOwnedFile(path string, nonblocking bool) (*os.File, error) {
	if err := rejectSymlinks(path); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	information, err := file.Stat()
	if err != nil || !information.Mode().IsRegular() || !verifyOwner(information, 0) {
		file.Close()
		return nil, UnsafePath
	}
	operation := unix.LOCK_EX
	if nonblocking {
		operation |= unix.LOCK_NB
	}
	if err := unix.Flock(int(file.Fd()), operation); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func configureProcess(command *exec.Cmd, databaseID uint, lock *os.File) error {
	userID, groupID, err := instanceCredentials(databaseID)
	if err != nil {
		return err
	}
	command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(userID), Gid: uint32(groupID)}, Pdeathsig: syscall.SIGTERM, Setpgid: true}
	command.ExtraFiles = []*os.File{lock}
	return nil
}

func processStartIdentity(processID int) string {
	contents, err := os.ReadFile("/proc/" + strconv.Itoa(processID) + "/stat")
	if err != nil {
		return ""
	}
	closing := strings.LastIndexByte(string(contents), ')')
	if closing < 0 {
		return ""
	}
	fields := strings.Fields(string(contents)[closing+1:])
	if len(fields) < 20 || fields[0] == "Z" || fields[0] == "X" {
		return ""
	}
	return fields[19]
}
