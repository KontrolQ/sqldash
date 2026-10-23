package runtime

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"sqldash/supervisor"
)

func managedPlatform() error {
	return nil
}

func postgresCredentials() (*supervisor.Credentials, error) {
	postgresUser, err := user.Lookup("postgres")
	if err != nil {
		return nil, errors.New("the dedicated PostgreSQL operating-system account is unavailable")
	}
	userID, userError := strconv.ParseUint(postgresUser.Uid, 10, 32)
	groupID, groupError := strconv.ParseUint(postgresUser.Gid, 10, 32)
	if userError != nil || groupError != nil || userID == 0 || groupID == 0 {
		return nil, errors.New("PostgreSQL needs a nonroot operating-system identity")
	}
	if os.Geteuid() != 0 && (uint32(os.Geteuid()) != uint32(userID) || uint32(os.Getegid()) != uint32(groupID)) {
		return nil, errors.New("the process cannot use the PostgreSQL operating-system identity")
	}
	return &supervisor.Credentials{UserID: uint32(userID), GroupID: uint32(groupID)}, nil
}

func configureCommand(command *exec.Cmd, credentials *supervisor.Credentials) error {
	if credentials == nil || credentials.UserID == 0 {
		return errors.New("PostgreSQL commands must not run as root")
	}
	return credentials.Configure(command)
}

func privateRegularFile(information os.FileInfo) bool {
	status, valid := information.Sys().(*syscall.Stat_t)
	return information.Mode().IsRegular() && valid && status.Nlink == 1
}

func setOwnership(path string, credentials *supervisor.Credentials) error {
	if credentials == nil {
		credentials = &supervisor.Credentials{UserID: uint32(os.Geteuid()), GroupID: uint32(os.Getegid())}
	}
	information, err := os.Lstat(path)
	if err != nil {
		return err
	}
	status, valid := information.Sys().(*syscall.Stat_t)
	if !valid {
		return errors.New("PostgreSQL storage ownership is unavailable")
	}
	if status.Uid == credentials.UserID && status.Gid == credentials.GroupID {
		return nil
	}
	if os.Geteuid() != 0 {
		return errors.New("PostgreSQL storage has incompatible ownership")
	}
	return os.Chown(path, int(credentials.UserID), int(credentials.GroupID))
}

func acquireClusterLock(path string) (*os.File, error) {
	if err := safePath(path); err != nil {
		return nil, err
	}
	descriptor, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	information, err := file.Stat()
	if err != nil || !information.Mode().IsRegular() || information.Mode().Perm() != 0o600 {
		file.Close()
		return nil, ErrClusterDamaged
	}
	if err = syscall.Flock(descriptor, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, ErrClusterLocked
	}
	if err = file.Sync(); err != nil {
		releaseClusterLock(file)
		return nil, err
	}
	if err = syncPathDirectory(filepath.Dir(path)); err != nil {
		releaseClusterLock(file)
		return nil, err
	}
	return file, nil
}

func releaseClusterLock(file *os.File) {
	if file != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}
}

func clusterRunning(dataPath string) (bool, error) {
	pidPath := filepath.Join(dataPath, "postmaster.pid")
	if err := regularFile(pidPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	contents, err := os.ReadFile(pidPath)
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(contents), "\n")
	if len(lines) < 2 || filepath.Clean(lines[1]) != filepath.Clean(dataPath) {
		return false, ErrClusterDamaged
	}
	processID, err := strconv.Atoi(lines[0])
	if err != nil || processID < 1 {
		return false, ErrClusterDamaged
	}
	err = syscall.Kill(processID, 0)
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, nil
	}
	return false, err
}

func fastShutdownSignal() os.Signal {
	return syscall.SIGINT
}

func syncPathDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
