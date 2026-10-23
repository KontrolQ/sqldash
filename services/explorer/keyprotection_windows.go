package explorer

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func protectIdentityKey(path string) error {
	user, userError := windows.GetCurrentProcessToken().GetTokenUser()
	if userError != nil {
		return userError
	}
	descriptor, descriptorError := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)")
	if descriptorError != nil {
		return descriptorError
	}
	permissions, _, permissionsError := descriptor.DACL()
	if permissionsError != nil {
		return permissionsError
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, permissions, nil)
}

func validateIdentityKey(path string) error {
	descriptor, descriptorError := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if descriptorError != nil {
		return descriptorError
	}
	control, _, controlError := descriptor.Control()
	if controlError != nil || control&windows.SE_DACL_PROTECTED == 0 {
		return errors.New("row signing key permissions are not protected")
	}
	permissions, _, permissionsError := descriptor.DACL()
	if permissionsError != nil || permissions == nil {
		return errors.New("row signing key permissions are unavailable")
	}
	user, userError := windows.GetCurrentProcessToken().GetTokenUser()
	if userError != nil {
		return userError
	}
	for index := uint32(0); index < uint32(permissions.AceCount); index++ {
		var entry *windows.ACCESS_ALLOWED_ACE
		if entryError := windows.GetAce(permissions, index, &entry); entryError != nil {
			return entryError
		}
		if entry.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return errors.New("unexpected row signing key permissions")
		}
		identity := (*windows.SID)(unsafe.Pointer(&entry.SidStart)).String()
		if identity != user.User.Sid.String() && identity != "S-1-5-18" && identity != "S-1-5-32-544" {
			return errors.New("row signing key permits another account")
		}
	}
	return nil
}
