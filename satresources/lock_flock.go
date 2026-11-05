//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package satresources

import (
	"errors"
	"os"
	"syscall"
)

// tryLock intenta obtener un flock exclusivo sin esperar.
func tryLock(p string) (unlock func(), ok bool, err error) {
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, false, err
	}
	fd := int(f.Fd())
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EINTR) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return func() {
		_ = syscall.Flock(fd, syscall.LOCK_UN)
		_ = f.Close()
	}, true, nil
}
