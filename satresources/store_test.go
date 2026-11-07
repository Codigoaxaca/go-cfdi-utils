package satresources_test

import (
	"context"
	"encoding/base64"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/codigoaxaca/go-cfdi-utils/internal/testresources"
	"github.com/codigoaxaca/go-cfdi-utils/internal/xmlsafe"
	"github.com/codigoaxaca/go-cfdi-utils/satresources"
	"github.com/codigoaxaca/go-cfdi-utils/satresources/embedded"
	"github.com/codigoaxaca/go-cfdi-utils/xmlschema"
)

const (
	cfdv40     = "http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd"
	cfdv40Key  = "www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd"
	leyendas   = "http://www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd"
	leyendaKey = "www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd"
	pagosKey   = "www.sat.gob.mx/sitio_internet/cfd/Pagos/Pagos20.xslt"
)

var ctx = context.Background()

func quiet() *slog.Logger { return slog.New(slog.DiscardHandler) }

func newStore(t *testing.T, sat *testresources.FakeSAT, cfg satresources.Config) *satresources.Store {
	t.Helper()
	if cfg.Dir == "" {
		cfg.Dir = t.TempDir()
	}
	if sat != nil {
		cfg.HTTPClient = sat.Client()
	}
	if cfg.Logger == nil {
		cfg.Logger = quiet()
	}
	if cfg.Fallback == nil {
		cfg.Fallback = emptyFallback{}
	}
	s := satresources.New(cfg)
	t.Cleanup(func() { satresources.WaitBackground(s) })
	return s
}

// emptyFallback evita usar el respaldo registrado al importar embedded.
type emptyFallback struct{}

func (emptyFallback) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

func TestPathDownloadsClosureAndRewritesSchemas(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if s.Version() != "" {
		t.Fatal("un almacén nuevo no debe tener versión")
	}
	p, err := s.Path(ctx, cfdv40)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `schemaLocation="../catalogos/catCFDI.xsd"`) {
		t.Fatalf("las importaciones no se reescribieron:\n%s", data[:400])
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(p), "../catalogos/catCFDI.xsd")); err != nil {
		t.Fatalf("falta la dependencia: %v", err)
	}
	if got := sat.Downloads.Load(); got != 3 {
		t.Fatalf("se esperaban 3 descargas (cfdv40, catCFDI, tdCFDI), hubo %d", got)
	}
	if s.Version() == "" {
		t.Fatal("se esperaba una versión activa")
	}
	// segunda vez: sin red
	if _, err := s.ReadFile(ctx, strings.Replace(cfdv40, "http:", "https:", 1)); err != nil {
		t.Fatal(err)
	}
	if got := sat.Requests.Load(); got != 3 {
		t.Fatalf("no debía haber más solicitudes, hubo %d", got)
	}
}

func TestPrepareAndRefreshWithoutChanges(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	now := time.Now()
	s := newStore(t, sat, satresources.Config{})
	satresources.SetClock(s, func() time.Time { return now })
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	st := s.Status()
	if st.Files < 60 || st.Version == "" || st.Fallback {
		t.Fatalf("estado inesperado: %+v", st)
	}
	downloads := sat.Downloads.Load()
	if err := s.Prepare(ctx); err != nil || sat.Downloads.Load() != downloads {
		t.Fatalf("Prepare no debe descargar de nuevo: %v", err)
	}
	now = now.Add(time.Hour)
	res, err := s.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed || res.Version != st.Version || res.Checked != st.Files {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if got := int(sat.NotModified.Load()); got != st.Files {
		t.Fatalf("se esperaban %d respuestas 304, hubo %d", st.Files, got)
	}
	if !s.Status().CheckedAt.Equal(now.UTC()) {
		t.Fatalf("no se actualizó la fecha de revisión: %v", s.Status().CheckedAt)
	}
}

func TestRefreshActivatesChanges(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	old := s.Version()
	oldPath, _ := s.Path(ctx, leyendas)
	sat.Set(leyendaKey, append(sat.Get(leyendaKey), []byte("<!-- cambio -->\n")...))
	res, err := s.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || res.Version == old || len(res.Updated) != 1 || res.Updated[0] != leyendas {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	p, err := s.Path(ctx, leyendas)
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(p); !strings.Contains(string(data), "<!-- cambio -->") {
		t.Fatal("no se usa la versión nueva")
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("la versión anterior debe conservarse: %v", err)
	}
}

func TestRefreshRejectsUnsupportedXSLT(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	old := s.Version()
	xsl := string(sat.Get(pagosKey))
	broken := strings.Replace(xsl, `<xsl:template match="pago20:Pagos">`, `<xsl:template match="pago20:Pagos"><xsl:choose/>`, 1)
	if broken == xsl {
		t.Fatal("no se encontró la plantilla a modificar")
	}
	sat.Set(pagosKey, []byte(broken))
	_, err := s.Refresh(ctx)
	if !errors.Is(err, satresources.ErrVerification) || !strings.Contains(err.Error(), "choose") {
		t.Fatalf("se esperaba error de verificación, se obtuvo %v", err)
	}
	st := s.Status()
	if st.Version != old || st.LastError == nil {
		t.Fatalf("debe seguir la versión anterior con el error registrado: %+v", st)
	}
	data, err := s.ReadFile(ctx, "http://"+pagosKey)
	if err != nil || strings.Contains(string(data), "xsl:choose") {
		t.Fatalf("se debe seguir usando el archivo anterior: %v", err)
	}
	// al corregirse en el SAT, la siguiente revisión funciona
	sat.Set(pagosKey, []byte(xsl))
	if _, err := s.Refresh(ctx); err != nil || s.Status().LastError != nil {
		t.Fatalf("la revisión debió funcionar: %v", err)
	}
}

func TestRefreshRejectsSchemaThatDoesNotCompile(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("xmllint"); err != nil {
		t.Skip("xmllint no está instalado")
	}
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	xsd := string(sat.Get(leyendaKey))
	broken := strings.Replace(xsd, `</xs:schema>`, `<xs:element name="Roto" type="noexiste:Tipo"/></xs:schema>`, 1)
	sat.Set(leyendaKey, []byte(broken))
	if _, err := s.Refresh(ctx); !errors.Is(err, satresources.ErrVerification) {
		t.Fatalf("se esperaba error de verificación, se obtuvo %v", err)
	}
}

// insertBeforeRoot agrega contenido justo antes de la raíz de un recurso.
func insertBeforeRoot(t *testing.T, data []byte, root, prolog string) []byte {
	t.Helper()
	out := strings.Replace(string(data), "<"+root, prolog+"<"+root, 1)
	if out == string(data) {
		t.Fatalf("no se encontró la raíz %s", root)
	}
	return []byte(out)
}

func utf7(s string) string {
	var raw []byte
	for _, u := range utf16.Encode([]rune(s)) {
		raw = append(raw, byte(u>>8), byte(u))
	}
	return "+" + base64.RawStdEncoding.EncodeToString(raw) + "-"
}

// Un recurso descargado con DTD no se activa: xmllint procesaría ese DTD al
// compilar el esquema y en cada validación. El DTD de la prueba no pide ningún
// archivo y el esquema compila: lo que debe rechazarlo es la verificación.
func TestRefreshRejectsResourceWithDTD(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ key, root string }{
		"xsd":  {leyendaKey, "xs:schema"},
		"xslt": {pagosKey, "xsl:stylesheet"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			sat := testresources.NewFakeSAT()
			s := newStore(t, sat, satresources.Config{})
			if err := s.Prepare(ctx); err != nil {
				t.Fatal(err)
			}
			old := s.Version()
			sat.Set(tc.key, insertBeforeRoot(t, sat.Get(tc.key), tc.root, `<!DOCTYPE `+tc.root+`>`))
			if _, err := s.Refresh(ctx); !errors.Is(err, xmlsafe.ErrDTD) {
				t.Fatalf("se esperaba xmlsafe.ErrDTD, se obtuvo %v", err)
			}
			if s.Version() != old {
				t.Fatal("no se debió activar otra versión")
			}
		})
	}
}

// Un DTD escondido en UTF-7 y uno dentro del esquema tampoco se aceptan.
func TestRefreshRejectsResourceHidingDTD(t *testing.T) {
	t.Parallel()
	dtd := `<!DOCTYPE xs:schema>`
	for name, tc := range map[string]struct {
		change func(xsd string) string
		want   error
	}{
		"utf7 con espacios": {func(xsd string) string {
			_, body, _ := strings.Cut(xsd, "?>")
			return `<?xml version="1.0" encoding = "UTF-7"?>` + utf7(dtd) + body
		}, xmlsafe.ErrDeclaration},
		"directiva dentro de la raíz": {func(xsd string) string {
			return strings.Replace(xsd, `</xs:schema>`, `<!ENTITY x "y"></xs:schema>`, 1)
		}, xmlsafe.ErrDTD},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			sat := testresources.NewFakeSAT()
			s := newStore(t, sat, satresources.Config{})
			if err := s.Prepare(ctx); err != nil {
				t.Fatal(err)
			}
			old := s.Version()
			sat.Set(leyendaKey, []byte(tc.change(string(sat.Get(leyendaKey)))))
			if _, err := s.Refresh(ctx); !errors.Is(err, tc.want) {
				t.Fatalf("se esperaba %v, se obtuvo %v", tc.want, err)
			}
			if s.Version() != old {
				t.Fatal("no se debió activar otra versión")
			}
		})
	}
}

func TestSchemaWithDTDCannotReadFiles(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("xmllint"); err != nil {
		t.Skip("xmllint no está instalado")
	}
	secret := filepath.Join(t.TempDir(), "secreto.env")
	if err := os.WriteFile(secret, []byte("DB_PASSWORD=hunter2-SECRETO\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sat := testresources.NewFakeSAT()
	sat.Set(cfdv40Key, insertBeforeRoot(t, sat.Get(cfdv40Key), "xs:schema", `<!DOCTYPE xs:schema [<!ENTITY % x SYSTEM "file://`+secret+`"> %x;]>`))
	s := newStore(t, sat, satresources.Config{})
	v, err := xmlschema.NewXmllint(s)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	err = v.ValidateSchemas(ctx, doc)
	if err == nil {
		t.Fatal("se esperaba un error por el esquema con DTD")
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("se filtró el contenido del archivo: %v", err)
	}
}

func TestRefreshFailsWhenDependencyIsMissing(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	old := s.Version()
	xsd := string(sat.Get(leyendaKey))
	sat.Set(leyendaKey, []byte(strings.Replace(xsd, `<xs:element`, `<xs:import namespace="urn:x" schemaLocation="nuevo.xsd"/><xs:element`, 1)))
	if _, err := s.Refresh(ctx); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("se esperaba error por el archivo faltante, se obtuvo %v", err)
	}
	if s.Version() != old {
		t.Fatal("no se debe activar una versión incompleta")
	}
}

func TestForceDownloadsEverything(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	old := s.Version()
	files := s.Status().Files
	before := sat.Downloads.Load()
	res, err := s.Refresh(ctx, satresources.Force)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || res.Version == old || len(res.Updated) != 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if got := int(sat.Downloads.Load() - before); got != files || sat.NotModified.Load() != 0 {
		t.Fatalf("se esperaban %d descargas completas, hubo %d (304: %d)", files, got, sat.NotModified.Load())
	}
}

func TestClear(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if _, err := s.Path(ctx, leyendas); err != nil {
		t.Fatal(err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if s.Version() != "" {
		t.Fatal("Clear debe quitar la versión activa")
	}
	if _, err := os.Stat(filepath.Join(s.Dir(), "snapshots")); !os.IsNotExist(err) {
		t.Fatalf("Clear debe borrar los archivos: %v", err)
	}
	before := sat.Downloads.Load()
	if _, err := s.Path(ctx, leyendas); err != nil || sat.Downloads.Load() == before {
		t.Fatalf("después de Clear se debe descargar de nuevo: %v", err)
	}
}

func TestOfflineUsesFallback(t *testing.T) {
	t.Parallel()
	s := newStore(t, nil, satresources.Config{Offline: true, Fallback: embedded.FS()})
	if _, err := s.Path(ctx, cfdv40); err != nil {
		t.Fatal(err)
	}
	st := s.Status()
	if !st.Fallback || st.Files < 60 || !st.NextCheck.IsZero() {
		t.Fatalf("estado inesperado: %+v", st)
	}
	if _, err := s.Refresh(ctx); !errors.Is(err, satresources.ErrOffline) {
		t.Fatalf("se esperaba ErrOffline, se obtuvo %v", err)
	}
	if _, err := s.Path(ctx, "http://www.sat.gob.mx/sitio_internet/cfd/nomina/nomina12.xsd"); !errors.Is(err, satresources.ErrOffline) {
		t.Fatalf("un recurso fuera del respaldo debe fallar con ErrOffline, se obtuvo %v", err)
	}
}

func TestOfflineWithoutFallback(t *testing.T) {
	t.Parallel()
	s := newStore(t, nil, satresources.Config{Offline: true})
	if _, err := s.Path(ctx, cfdv40); !errors.Is(err, satresources.ErrOffline) {
		t.Fatalf("se esperaba ErrOffline, se obtuvo %v", err)
	}
}

func TestFallbackWhenSATIsDownThenReplaced(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	sat.Fail.Store(true)
	s := newStore(t, sat, satresources.Config{Fallback: embedded.FS(), MaxAge: -1})
	if _, err := s.Path(ctx, cfdv40); err != nil {
		t.Fatal(err)
	}
	st := s.Status()
	if !st.Fallback || st.LastError == nil || !st.CheckedAt.IsZero() {
		t.Fatalf("se esperaba el respaldo con el error registrado: %+v", st)
	}
	sat.Fail.Store(false)
	res, err := s.Refresh(ctx)
	if err != nil || !res.Changed {
		t.Fatalf("Refresh debe reemplazar el respaldo: %+v %v", res, err)
	}
	if st := s.Status(); st.Fallback || st.LastError != nil {
		t.Fatalf("estado inesperado: %+v", st)
	}
}

func TestSATDownWithoutFallback(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	sat.Fail.Store(true)
	s := newStore(t, sat, satresources.Config{})
	if _, err := s.Path(ctx, cfdv40); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("se esperaba el error de descarga, se obtuvo %v", err)
	}
}

func TestAutomaticRefresh(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	var mu sync.Mutex
	now := time.Now()
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	advance := func(d time.Duration) { mu.Lock(); now = now.Add(d); mu.Unlock() }
	s := newStore(t, sat, satresources.Config{MaxAge: 24 * time.Hour, RetryInterval: time.Hour})
	satresources.SetClock(s, clock)
	satresources.SetPointerTTL(s, 0)
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	old := s.Version()
	sat.Set(leyendaKey, append(sat.Get(leyendaKey), []byte("<!-- cambio -->\n")...))

	// antes de MaxAge no se revisa
	advance(23 * time.Hour)
	requests := sat.Requests.Load()
	if _, err := s.Path(ctx, cfdv40); err != nil {
		t.Fatal(err)
	}
	satresources.WaitBackground(s)
	if sat.Requests.Load() != requests || s.Version() != old {
		t.Fatal("no debía revisar antes de MaxAge")
	}

	// después de MaxAge se revisa en segundo plano y se activa la versión nueva
	advance(2 * time.Hour)
	if _, err := s.Path(ctx, cfdv40); err != nil {
		t.Fatal(err)
	}
	satresources.WaitBackground(s)
	if s.Version() == old {
		t.Fatal("no se activó la versión nueva")
	}

	// si la revisión falla, no se reintenta antes de RetryInterval
	advance(25 * time.Hour)
	sat.Fail.Store(true)
	cur := s.Version()
	if _, err := s.Path(ctx, cfdv40); err != nil {
		t.Fatal(err)
	}
	satresources.WaitBackground(s)
	if s.Status().LastError == nil || s.Version() != cur {
		t.Fatal("se esperaba el error registrado y la misma versión")
	}
	requests = sat.Requests.Load()
	advance(30 * time.Minute)
	if _, err := s.Path(ctx, cfdv40); err != nil {
		t.Fatal(err)
	}
	satresources.WaitBackground(s)
	if sat.Requests.Load() != requests {
		t.Fatal("no debía reintentar antes de RetryInterval")
	}
}

func TestAllowHost(t *testing.T) {
	t.Parallel()
	s := newStore(t, testresources.NewFakeSAT(), satresources.Config{})
	if s.Allowed("http://example.com/addenda.xsd") || !s.Allowed(cfdv40) || s.Allowed("ftp://www.sat.gob.mx/x.xsd") {
		t.Fatal("Allowed inesperado")
	}
	if _, err := s.Path(ctx, "http://example.com/addenda.xsd"); !errors.Is(err, satresources.ErrNotAllowed) {
		t.Fatalf("se esperaba ErrNotAllowed, se obtuvo %v", err)
	}
}

func TestSharedDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sat1, sat2 := testresources.NewFakeSAT(), testresources.NewFakeSAT()
	s1 := newStore(t, sat1, satresources.Config{Dir: dir})
	s2 := newStore(t, sat2, satresources.Config{Dir: dir})
	satresources.SetPointerTTL(s2, 0)
	if err := s1.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Path(ctx, cfdv40); err != nil || sat2.Requests.Load() != 0 {
		t.Fatalf("el segundo proceso debe usar lo descargado por el primero: %v", err)
	}
	sat1.Set(leyendaKey, append(sat1.Get(leyendaKey), []byte("<!-- cambio -->\n")...))
	if _, err := s1.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if s2.Version() != s1.Version() {
		t.Fatal("el segundo proceso debe ver la versión nueva")
	}
}

func TestExportAndUseAsFallback(t *testing.T) {
	t.Parallel()
	s := newStore(t, testresources.NewFakeSAT(), satresources.Config{})
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := s.Export(out); err != nil {
		t.Fatal(err)
	}
	offline := newStore(t, nil, satresources.Config{Offline: true, Fallback: os.DirFS(out)})
	if _, err := offline.Path(ctx, satresources.XsltCfdi40); err != nil {
		t.Fatal(err)
	}
	if offline.Status().Files != s.Status().Files {
		t.Fatal("el respaldo exportado debe tener todos los archivos")
	}
}

func TestConcurrentPath(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	var wg sync.WaitGroup
	var failures atomic.Int32
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, r := range []string{cfdv40, satresources.XsltCfdi40, leyendas} {
				if _, err := s.ReadFile(ctx, r); err != nil {
					failures.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	if failures.Load() != 0 {
		t.Fatal("hubo errores")
	}
	if int(sat.Downloads.Load()) != s.Status().Files {
		t.Fatalf("cada archivo se debe descargar una vez: %d descargas, %d archivos", sat.Downloads.Load(), s.Status().Files)
	}
}

func TestStatusString(t *testing.T) {
	t.Parallel()
	s := newStore(t, testresources.NewFakeSAT(), satresources.Config{})
	if !strings.Contains(s.Status().String(), "sin recursos guardados") {
		t.Fatal(s.Status().String())
	}
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	out := s.Status().String()
	for _, want := range []string{"versión: ", "origen: SAT", "próxima revisión: "} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q en:\n%s", want, out)
		}
	}
}

// Un XSD nuevo que se descarga bajo demanda también se compila antes de usarlo.
func TestOnDemandSchemaIsCompiled(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("xmllint"); err != nil {
		t.Skip("xmllint no está instalado")
	}
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if _, err := s.Path(ctx, leyendas); err != nil {
		t.Fatal(err)
	}
	old := s.Version()
	sat.Set("www.sat.gob.mx/sitio_internet/cfd/nuevo/nuevo.xsd", []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="Roto" type="noexiste:Tipo"/></xs:schema>`))
	_, err := s.Path(ctx, "http://www.sat.gob.mx/sitio_internet/cfd/nuevo/nuevo.xsd")
	if !errors.Is(err, satresources.ErrVerification) {
		t.Fatalf("se esperaba error de verificación, se obtuvo %v", err)
	}
	if s.Version() != old || s.Status().Files != 1 {
		t.Fatalf("el XSD inválido no debe agregarse: %+v", s.Status())
	}
	matches, _ := filepath.Glob(filepath.Join(s.Dir(), "snapshots", "*", "files", "www.sat.gob.mx", "sitio_internet", "cfd", "nuevo", "nuevo.xsd"))
	if len(matches) != 0 {
		t.Fatalf("no deben quedar archivos rechazados: %v", matches)
	}
}

// Una inclusión circular no evita la verificación.
func TestIncludeCycleIsRejected(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	head := `<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">`
	sat.Set("www.sat.gob.mx/ciclo/a.xslt", []byte(head+`<xsl:include href="b.xslt"/><xsl:template match="/">a</xsl:template></xsl:stylesheet>`))
	sat.Set("www.sat.gob.mx/ciclo/b.xslt", []byte(head+`<xsl:include href="a.xslt"/><xsl:template name="b"><xsl:choose/></xsl:template></xsl:stylesheet>`))
	if _, err := s.Path(ctx, "http://www.sat.gob.mx/ciclo/a.xslt"); !errors.Is(err, satresources.ErrVerification) {
		t.Fatalf("se esperaba error de verificación, se obtuvo %v", err)
	}
}

// Una versión que acaba de dejar de estar activa no se borra aunque sea antigua.
func TestPruneKeepsRecentlyReplacedVersions(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if err := s.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	dirOf := func(id string) string { return filepath.Join(s.Dir(), "snapshots", id) }
	age := func(id string) {
		old := time.Now().Add(-48 * time.Hour)
		if err := os.Chtimes(dirOf(id), old, old); err != nil {
			t.Fatal(err)
		}
	}
	a := s.Version()
	age(a) // activa desde hace días
	for i := 0; i < 2; i++ {
		if _, err := s.Refresh(ctx, satresources.Force); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(dirOf(a)); err != nil {
		t.Fatalf("la versión reemplazada hace segundos no debe borrarse: %v", err)
	}
	age(a) // reemplazada hace más de una hora
	if _, err := s.Refresh(ctx, satresources.Force); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dirOf(a)); !os.IsNotExist(err) {
		t.Fatalf("la versión antigua debió borrarse: %v", err)
	}
}

// Las redirecciones solo pueden ir a servidores permitidos.
func TestRedirectToDisallowedHost(t *testing.T) {
	t.Parallel()
	sat := testresources.NewFakeSAT()
	sat.Redirect(leyendaKey, "https://example.com/leyendasFisc.xsd")
	s := newStore(t, sat, satresources.Config{})
	if _, err := s.Path(ctx, leyendas); !errors.Is(err, satresources.ErrNotAllowed) {
		t.Fatalf("se esperaba ErrNotAllowed, se obtuvo %v", err)
	}
}

// Si xmllint no termina bien, la versión no se activa.
func TestXmllintFailureRejectsVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("usa un script de shell")
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "xmllint"), []byte("#!/bin/sh\nkill -9 $$\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	sat := testresources.NewFakeSAT()
	s := newStore(t, sat, satresources.Config{})
	if _, err := s.Path(ctx, leyendas); !errors.Is(err, satresources.ErrVerification) {
		t.Fatalf("se esperaba error de verificación, se obtuvo %v", err)
	}
	if s.Version() != "" {
		t.Fatal("no se debe activar una versión sin verificar")
	}
}
