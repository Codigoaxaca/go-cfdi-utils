package satresources

import (
	"context"
	"time"
)

// Ganchos para las pruebas externas.

func WaitBackground(s *Store)                   { s.bg.Wait() }
func SetClock(s *Store, now func() time.Time)   { s.now = now }
func SetPointerTTL(s *Store, ttl time.Duration) { s.pointerTTL = ttl }

func LockContext(ctx context.Context, s *Store) (func(), error) { return s.lock(ctx) }
