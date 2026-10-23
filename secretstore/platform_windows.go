package secretstore

import (
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

func lockFile(file *os.File) error {
	return windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &windows.Overlapped{})
}

func unlockFile(file *os.File) {
	_ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &windows.Overlapped{})
}

func privateDescriptor(directory bool) (*windows.SECURITY_DESCRIPTOR, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	inheritance := ""
	if directory {
		inheritance = "OICI"
	}
	return windows.SecurityDescriptorFromString("D:P(A;" + inheritance + ";FA;;;SY)(A;" + inheritance + ";FA;;;" + user.User.Sid.String() + ")")
}

func protectPath(path string, directory bool) error {
	descriptor, err := privateDescriptor(directory)
	if err != nil {
		return err
	}
	accessList, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, accessList, nil)
}

func privateMode(mode os.FileMode, directory bool) bool {
	return true
}

func validatePrivateLinks(path string, information os.FileInfo) error {
	encodedPath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(encodedPath, windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	var details windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(handle, &details); err != nil {
		return err
	}
	if details.NumberOfLinks != 1 {
		return ErrUnsafePath
	}
	return nil
}

func validateProtection(path string, directory bool) error {
	expected, err := privateDescriptor(directory)
	if err != nil {
		return err
	}
	actual, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := actual.Owner()
	if err != nil {
		return err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	if !owner.Equals(user.User.Sid) {
		return ErrUnsafePath
	}
	control, _, err := actual.Control()
	if err != nil {
		return err
	}
	actualText := actual.String()
	expectedText := expected.String()
	actualStart := strings.IndexByte(actualText, '(')
	expectedStart := strings.IndexByte(expectedText, '(')
	if control&windows.SE_DACL_PROTECTED == 0 || actualStart < 0 || expectedStart < 0 || actualText[actualStart:] != expectedText[expectedStart:] {
		return ErrUnsafePath
	}
	return nil
}

func syncDirectory(path string) error {
	return nil
}
