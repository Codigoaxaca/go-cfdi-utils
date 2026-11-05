package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUsageAndStatus(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	if code := run(nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "uso:") {
		t.Fatalf("sin comando: %d %q", code, errOut.String())
	}
	errOut.Reset()
	if code := run([]string{"-dir", dir, "nada"}, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "comando desconocido") {
		t.Fatalf("comando desconocido: %d %q", code, errOut.String())
	}
	if code := run([]string{"-dir", dir, "status"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "sin recursos guardados") {
		t.Fatalf("status: %d %q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"-dir", dir, "clear"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "recursos borrados") {
		t.Fatalf("clear: %d %q", code, out.String())
	}
	if code := run([]string{"-dir", dir, "export", t.TempDir()}, &out, &errOut); code != 1 {
		t.Fatalf("export sin recursos debe fallar: %d", code)
	}
	if code := run([]string{"-dir", dir, "update", "-force", "extra"}, &out, &errOut); code != 2 {
		t.Fatalf("argumentos de más: %d", code)
	}
}
