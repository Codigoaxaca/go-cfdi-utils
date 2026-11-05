//go:build windows

package satresources

import (
	"errors"
	"syscall"
)

const errorSharingViolation syscall.Errno = 32

// tryLock abre el archivo sin compartirlo con nadie; Windows cierra el
// archivo (y libera el candado) si el proceso termina.
func tryLock(p string) (unlock func(), ok bool, err error) {
	name, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return nil, false, err
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if errors.Is(err, errorSharingViolation) || errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return func() { _ = syscall.CloseHandle(h) }, true, nil
}
