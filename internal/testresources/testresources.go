// Package testresources configura, para las pruebas, un almacén de recursos
// del SAT sin conexión que usa el respaldo incluido, de modo que las pruebas
// no dependen de la red ni de la caché del usuario.
package testresources

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/satresources"
	"github.com/codigoaxaca/go-cfdi-utils/satresources/embedded"
)

// Setup reemplaza satresources.Default() por un almacén en un directorio
// temporal en modo sin conexión. Devuelve una función que borra el directorio.
func Setup() (cleanup func()) {
	dir, err := os.MkdirTemp("", "go-cfdi-utils-test-*")
	if err != nil {
		panic(err)
	}
	satresources.SetDefault(satresources.New(satresources.Config{
		Dir:      dir,
		Offline:  true,
		Fallback: embedded.FS(),
		Logger:   slog.New(slog.DiscardHandler),
	}))
	return func() { _ = os.RemoveAll(dir) }
}

// Main se usa desde TestMain: func TestMain(m *testing.M) { testresources.Main(m) }
func Main(m *testing.M) {
	cleanup := Setup()
	code := m.Run()
	cleanup()
	if code != 0 {
		fmt.Fprintln(os.Stderr, "testresources: pruebas con recursos del respaldo incluido")
	}
	os.Exit(code)
}
