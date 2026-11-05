//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd || windows)

package satresources

import (
	"errors"
	"io/fs"
	"os"
	"time"
)

// En plataformas sin flock se usa un archivo creado con O_EXCL. Un candado
// sin renovar por más de un minuto se considera abandonado.
func tryLock(p string) (unlock func(), ok bool, err error) {
	excl := p + ".excl"
	f, err := os.OpenFile(excl, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return nil, false, err
		}
		if fi, err := os.Stat(excl); err == nil && time.Since(fi.ModTime()) > time.Minute {
			_ = os.Remove(excl)
		}
		return nil, false, nil
	}
	_ = f.Close()
	done, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				now := time.Now()
				_ = os.Chtimes(excl, now, now)
			}
		}
	}()
	return func() {
		close(done)
		<-stopped
		_ = os.Remove(excl)
	}, true, nil
}
