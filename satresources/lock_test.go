package satresources_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/satresources"
)

func lockWithin(dir string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	unlock, err := satresources.LockContext(ctx, satresources.New(satresources.Config{Dir: dir}))
	if err == nil {
		unlock()
	}
	return err
}

func TestLockIsExclusive(t *testing.T) {
	dir := t.TempDir()
	unlock, err := satresources.LockContext(context.Background(), satresources.New(satresources.Config{Dir: dir}))
	if err != nil {
		t.Fatal(err)
	}
	if err := lockWithin(dir, 200*time.Millisecond); err == nil {
		t.Fatal("el candado debe ser exclusivo")
	}
	unlock()
	if err := lockWithin(dir, time.Second); err != nil {
		t.Fatalf("después de liberarlo se debe poder obtener: %v", err)
	}
}

// Proceso auxiliar de TestLockReleasedWhenProcessDies: toma el candado y espera.
func TestLockHelperProcess(t *testing.T) {
	dir := os.Getenv("SATRESOURCES_LOCK_HELPER")
	if dir == "" {
		t.Skip("solo se usa como proceso auxiliar")
	}
	if _, err := satresources.LockContext(context.Background(), satresources.New(satresources.Config{Dir: dir})); err != nil {
		os.Exit(2)
	}
	fmt.Println("candado")
	time.Sleep(time.Minute)
	os.Exit(0)
}

// Si el proceso que tiene el candado muere, el sistema lo libera de inmediato.
func TestLockReleasedWhenProcessDies(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockHelperProcess$")
	cmd.Env = append(os.Environ(), "SATRESOURCES_LOCK_HELPER="+dir)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || line != "candado\n" {
		_ = cmd.Process.Kill()
		t.Fatalf("el proceso auxiliar no tomó el candado: %q %v", line, err)
	}
	if err := lockWithin(dir, 200*time.Millisecond); err == nil {
		_ = cmd.Process.Kill()
		t.Fatal("el candado del otro proceso debe respetarse")
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	if err := lockWithin(dir, 2*time.Second); err != nil {
		t.Fatalf("el candado debió liberarse al morir el proceso: %v", err)
	}
}
