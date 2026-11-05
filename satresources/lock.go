package satresources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lock obtiene el candado entre procesos sobre <Dir>/.lock. Se usa el
// candado del sistema operativo (flock en Linux/macOS/BSD, apertura
// exclusiva en Windows), que el sistema libera si el proceso termina sin
// liberarlo. El archivo .lock nunca se borra.
func (s *Store) lock(ctx context.Context) (func(), error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, fmt.Errorf("satresources: %w", err)
	}
	p := filepath.Join(s.dir, ".lock")
	for {
		unlock, ok, err := tryLock(p)
		if err != nil {
			return nil, fmt.Errorf("satresources: candado %s: %w", p, err)
		}
		if ok {
			return unlock, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("satresources: esperando el candado %s: %w", p, ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}
