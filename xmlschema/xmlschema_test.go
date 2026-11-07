package xmlschema

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/codigoaxaca/go-cfdi-utils/internal/testresources"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

func TestMain(m *testing.M) { testresources.Main(m) }

func TestSchemaLocations(t *testing.T) {
	root, err := nodes.ParseXMLFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	locs := SchemaLocations(root)
	if len(locs) < 2 || locs[0].Namespace != "http://www.sat.gob.mx/cfd/4" {
		t.Fatalf("ubicaciones inesperadas %+v", locs)
	}
}

func TestValidateSchemas(t *testing.T) {
	v, err := NewXmllint(nil)
	if errors.Is(err, ErrXmllintNotFound) {
		t.Skip("xmllint no está instalado")
	}
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := v.ValidateSchemas(context.Background(), data); err != nil {
		t.Fatalf("el CFDI real debería ser válido: %v", err)
	}
	invalid := strings.Replace(string(data), `Exportacion="01"`, `Exportacion="99"`, 1)
	err = v.ValidateSchemas(context.Background(), []byte(invalid))
	var schemaErr *Error
	if !errors.As(err, &schemaErr) || !strings.Contains(err.Error(), "Exportacion") {
		t.Fatalf("se esperaba un error de esquema por Exportacion, se obtuvo %v", err)
	}
	t.Log(err)
}

const testSecret = "DB_PASSWORD=hunter2-SECRETO"

func utf7(s string) string {
	var raw []byte
	for _, u := range utf16.Encode([]rune(s)) {
		raw = append(raw, byte(u>>8), byte(u))
	}
	return "+" + base64.RawStdEncoding.EncodeToString(raw) + "-"
}

// fileReadAttacks arma variantes de un CFDI real cuyo DTD intenta leer path.
func fileReadAttacks(t *testing.T, path string) map[string][]byte {
	t.Helper()
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, body, found := strings.Cut(string(data), "?>")
	if !found {
		t.Fatal("el CFDI de prueba no tiene declaración XML")
	}
	dtd := `<!DOCTYPE cfdi:Comprobante [<!ENTITY % x SYSTEM "file://` + path + `"> %x;]>`
	hidden := utf7(dtd)
	return map[string][]byte{
		"dtd":                 []byte(`<?xml version="1.0" encoding="UTF-8"?>` + dtd + body),
		"utf7":                []byte(`<?xml version="1.0" encoding="UTF-7"?>` + hidden + body),
		"utf7 con espacios":   []byte(`<?xml version="1.0" encoding = "UTF-7"?>` + hidden + body),
		"utf7 espacio luego":  []byte(`<?xml version="1.0" encoding= "UTF-7"?>` + hidden + body),
		"utf7 en comentario":  []byte(`<?xml version="1.0" encoding = "UTF-7"?><!-- x ` + utf7(`-->`) + hidden + utf7(`<!-- y`) + ` -->` + body),
		"utf7 en instrucción": []byte(`<?xml version="1.0" encoding = "UTF-7"?><?p ` + utf7(`?>`) + hidden + utf7(`<?q`) + ` ?>` + body),
	}
}

// El XML con DTD no debe llegar nunca a xmllint: este lee el DTD por sí mismo
// y no hay opción para impedir que abra archivos locales.
func TestDangerousXMLNeverReachesXmllint(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "xmllint-ejecutado")
	v := &XmllintValidator{Executable: writeScript(t, `: > "`+marker+`"`)}

	for name, doc := range fileReadAttacks(t, "/etc/passwd") {
		if err := v.ValidateSchemas(context.Background(), doc); err == nil {
			t.Errorf("%s: se esperaba un error", name)
		}
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("%s: el documento llegó a xmllint", name)
		}
	}

	// Comprobación del propio mecanismo: un XML normal sí llega a xmllint.
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := v.ValidateSchemas(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("el documento normal debió llegar a xmllint")
	}
}

// Con el xmllint real: ningún documento consigue sacar el contenido del archivo.
func TestXMLCannotReadFiles(t *testing.T) {
	v, err := NewXmllint(nil)
	if errors.Is(err, ErrXmllintNotFound) {
		t.Skip("xmllint no está instalado")
	}
	secret := filepath.Join(t.TempDir(), "secreto.env")
	if err := os.WriteFile(secret, []byte(testSecret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, doc := range fileReadAttacks(t, secret) {
		err := v.ValidateSchemas(context.Background(), doc)
		if err == nil {
			t.Errorf("%s: se esperaba un error", name)
		} else if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("%s: se filtró el contenido del archivo: %v", name, err)
		}
	}
}

func TestDTDErrorIsIdentifiable(t *testing.T) {
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	v := &XmllintValidator{Executable: writeScript(t, "exit 0")}
	doc := strings.Replace(string(data), "<cfdi:Comprobante", `<!DOCTYPE cfdi:Comprobante><cfdi:Comprobante`, 1)
	if err := v.ValidateSchemas(context.Background(), []byte(doc)); !errors.Is(err, nodes.ErrDTDNotAllowed) {
		t.Fatalf("se esperaba ErrDTDNotAllowed, se obtuvo %v", err)
	}
}

func TestValidateSchemasStopsHungXmllint(t *testing.T) {
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	// sh se queda esperando a sleep, que además conserva abierta la salida de
	// errores: terminar solo a sh no bastaría para liberar la llamada.
	hung := writeScript(t, "sleep 30\nexit 0")

	const limit = 15 * time.Second
	deadline := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 100*time.Millisecond)
	}
	cancelled := func() (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(100*time.Millisecond, cancel)
		return ctx, cancel
	}
	background := func() (context.Context, context.CancelFunc) {
		return context.Background(), func() {}
	}
	for name, tc := range map[string]struct {
		timeout time.Duration
		ctx     func() (context.Context, context.CancelFunc)
		want    error
	}{
		"límite del validador": {100 * time.Millisecond, background, context.DeadlineExceeded},
		"plazo de quien llama": {0, deadline, context.DeadlineExceeded},
		"cancelación":          {0, cancelled, context.Canceled},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := &XmllintValidator{Executable: hung, Timeout: tc.timeout}
			ctx, cancel := tc.ctx()
			defer cancel()
			start := time.Now()
			err := v.ValidateSchemas(ctx, data)
			if elapsed := time.Since(start); elapsed > limit {
				t.Fatalf("tardó %v en detenerse", elapsed)
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("se esperaba %v, se obtuvo %v", tc.want, err)
			}
			var schemaErr *Error
			if errors.As(err, &schemaErr) {
				t.Fatalf("un xmllint interrumpido no significa que el XML sea inválido: %v", err)
			}
		})
	}
}

func TestDefaultTimeout(t *testing.T) {
	if got := (&XmllintValidator{}).timeout(); got != DefaultTimeout {
		t.Fatalf("sin Timeout se esperaba %v, se obtuvo %v", DefaultTimeout, got)
	}
	if got := (&XmllintValidator{Timeout: -time.Second}).timeout(); got != DefaultTimeout {
		t.Fatalf("con Timeout negativo se esperaba %v, se obtuvo %v", DefaultTimeout, got)
	}
	if got := (&XmllintValidator{Timeout: time.Second}).timeout(); got != time.Second {
		t.Fatalf("se esperaba 1s, se obtuvo %v", got)
	}
}

// writeScript crea un ejecutable de shell que reemplaza a xmllint.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("requiere un shell POSIX")
	}
	path := filepath.Join(t.TempDir(), "xmllint")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
