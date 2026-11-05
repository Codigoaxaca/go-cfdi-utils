package cadenaorigen

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/codigoaxaca/go-cfdi-utils/internal/testresources"
	"github.com/codigoaxaca/go-cfdi-utils/satresources"
)

func TestMain(m *testing.M) { testresources.Main(m) }

func TestGoldens(t *testing.T) {
	files, err := filepath.Glob("../testdata/cfdi/*.xml")
	if err != nil {
		t.Fatal(err)
	}
	extras, err := filepath.Glob("testdata/xml/*.xml")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, "../testdata/retenciones/retenciones20.xml")
	files = append(files, extras...)
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Build(data)
			if err != nil {
				t.Fatal(err)
			}
			gold := filepath.Join("testdata/golden", strings.TrimSuffix(filepath.Base(file), ".xml")+".txt")
			want, err := os.ReadFile(gold)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("got  %q\nwant %q", got, want)
			}
			if strings.HasPrefix(filepath.Base(file), "tfd-") {
				other, err := BuildTFD(data)
				if err != nil || other != got {
					t.Fatalf("BuildTFD: %q %v", other, err)
				}
			}
			if strings.HasPrefix(filepath.Base(file), "cfdi33-") {
				ref := strings.TrimSuffix(file, ".xml") + "-cadenaorigen.txt"
				want, err := os.ReadFile(ref)
				if err != nil {
					t.Fatal(err)
				}
				// The valid fixture's certificate was replaced after the PHP golden
				// was generated. Compare its original certificate in memory only;
				// the unmodified XML is checked against xsltproc above.
				phpData := data
				if filepath.Base(file) == "cfdi33-valid.xml" {
					phpData = []byte(strings.Replace(string(data), `NoCertificado="30001000000400002434"`, `NoCertificado="30001000000500003416"`, 1))
				}
				phpGot, err := Build(phpData)
				if err != nil || phpGot != string(want) {
					t.Fatalf("PHP reference differs: got %q want %q: %v", phpGot, want, err)
				}
			}
		})
	}
}
func TestLocationFor(t *testing.T) {
	for _, tc := range []struct{ ns, name, version, loc string }{{"cfd/4", "Comprobante", "4.0", XsltCfdi40}, {"cfd/3", "Comprobante", "3.3", XsltCfdi33}, {"TimbreFiscalDigital", "TimbreFiscalDigital", "1.1", XsltTfd11}, {"esquemas/retencionpago/2", "Retenciones", "2.0", XsltRetenciones20}} {
		for _, attr := range []string{"Version", "version"} {
			data := []byte(fmt.Sprintf(`<%s xmlns="http://www.sat.gob.mx/%s" %s="%s"/>`, tc.name, tc.ns, attr, tc.version))
			got, err := LocationFor(data)
			if err != nil || got != tc.loc {
				t.Fatalf("%s: %s %v", data, got, err)
			}
		}
	}
	for _, data := range []string{`<Comprobante Version="4.0"/>`, `<x:Comprobante xmlns:x="http://www.sat.gob.mx/cfd/4" Version="3.3"/>`, `<x/>`} {
		if _, err := LocationFor([]byte(data)); !errors.Is(err, ErrUnsupportedDocument) {
			t.Fatalf("%s: %v", data, err)
		}
	}
	for _, data := range []string{"", `<x>`, `<x/><y/>`, `text<x/>`} {
		if _, err := LocationFor([]byte(data)); err == nil {
			t.Fatalf("accepted %q", data)
		}
	}
	if _, err := BuildTFD([]byte(`<Comprobante xmlns="http://www.sat.gob.mx/cfd/4" Version="4.0"/>`)); !errors.Is(err, ErrUnsupportedDocument) {
		t.Fatal(err)
	}
}
func TestConcurrent(t *testing.T) {
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	want, err := Build(data)
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuilder()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				got, err := b.Build(data, XsltCfdi40)
				if err != nil || got != want {
					t.Errorf("got %q, %v", got, err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

const styleHead = `<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:a="urn:a" version="2.0">`

func TestFSIncludesNamespacesAndBuiltins(t *testing.T) {
	files := fstest.MapFS{
		"example.test/main.xslt": {Data: []byte(styleHead + `<xsl:include href="inc.xslt"/><xsl:include href="https://example.test/inc.xslt"/><xsl:template match="/"><xsl:apply-templates select="/a:Root"/></xsl:template><xsl:template match="a:Child">last</xsl:template></xsl:stylesheet>`)},
		"example.test/inc.xslt":  {Data: []byte(styleHead + `<xsl:template match="a:Child">first</xsl:template></xsl:stylesheet>`)},
	}
	b := NewBuilder(WithFS(files))
	got, err := b.Build([]byte("<z:Root xmlns:z=\"urn:a\">\n <z:Child/> text <Unknown>nested</Unknown>\n</z:Root>"), "http://example.test/main.xslt")
	if err != nil || got != "\n last text nested\n" {
		t.Fatalf("got %q: %v", got, err)
	}
	// WithFS uses only the given files, with HTTP/HTTPS sharing one key.
	override := fstest.MapFS{"www.sat.gob.mx/sitio_internet/cfd/4/cadenaoriginal_4_0/cadenaoriginal_4_0.xslt": {Data: []byte(styleHead + `<xsl:template match="/">override</xsl:template></xsl:stylesheet>`)}}
	got, err = NewBuilder(WithFS(override)).Build([]byte(`<x/>`), strings.Replace(XsltCfdi40, "http:", "https:", 1))
	if err != nil || got != "override" {
		t.Fatalf("%q %v", got, err)
	}
}
func TestAllEntryPointsCompile(t *testing.T) {
	b := NewBuilder()
	for _, loc := range []string{XsltCfdi40, XsltCfdi33, XsltTfd11, XsltRetenciones20} {
		if _, err := b.compile(loc); err != nil {
			t.Errorf("%s: %v", loc, err)
		}
	}
}

// El constructor usa la versión nueva en cuanto el almacén la activa.
func TestBuilderFollowsStoreVersion(t *testing.T) {
	sat := testresources.NewFakeSAT()
	store := satresources.New(satresources.Config{Dir: t.TempDir(), HTTPClient: sat.Client(), Logger: slog.New(slog.DiscardHandler)})
	b := NewBuilder(WithStore(store))
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	before, err := b.Build(data, XsltCfdi40)
	if err != nil {
		t.Fatal(err)
	}
	// El SAT cambia la plantilla de Concepto: ya no incluye la descripción.
	key := "www.sat.gob.mx/sitio_internet/cfd/4/cadenaoriginal_4_0/cadenaoriginal_4_0.xslt"
	xsl := string(sat.Get(key))
	changed := strings.Replace(xsl, `<xsl:with-param name="valor" select="./@Descripcion"/>`, `<xsl:with-param name="valor" select="./@Unidad"/>`, 1)
	if changed == xsl {
		t.Fatal("no se encontró la plantilla a modificar")
	}
	sat.Set(key, []byte(changed))
	res, err := store.Refresh(context.Background())
	if err != nil || !res.Changed {
		t.Fatalf("Refresh: %+v %v", res, err)
	}
	after, err := b.Build(data, XsltCfdi40)
	if err != nil {
		t.Fatal(err)
	}
	if after == before {
		t.Fatal("el constructor siguió usando la versión anterior")
	}
}

func TestErrors(t *testing.T) {
	for _, xslt := range []string{styleHead + `<xsl:template match="/"><xsl:choose/></xsl:template></xsl:stylesheet>`, styleHead + `<xsl:template match="/"><xsl:call-template name="missing"/></xsl:template></xsl:stylesheet>`, `not XML`} {
		b := NewBuilder(WithFS(fstest.MapFS{"test.invalid/a": {Data: []byte(xslt)}}))
		if _, err := b.Build([]byte(`<x/>`), "http://test.invalid/a"); err == nil {
			t.Fatal("expected error")
		}
	}
}

func TestFileNamespacesAndPaths(t *testing.T) {
	files := fstest.MapFS{
		"example.test/main": {Data: []byte(styleHead + `<xsl:include href="inc"/><xsl:template match="/"><xsl:apply-templates select="/a:Root"/></xsl:template><xsl:template match="a:Root"><xsl:apply-templates select="./*"/></xsl:template></xsl:stylesheet>`)},
		"example.test/inc":  {Data: []byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:a="urn:b" version="2.0"><xsl:template match="a:Child"><xsl:for-each select=".//a:Item"><xsl:value-of select="./@Año"/></xsl:for-each><xsl:if test="./@version='1.1'">yes</xsl:if></xsl:template></xsl:stylesheet>`)},
	}
	b := NewBuilder(WithFS(files))
	got, err := b.Build([]byte(`<Root xmlns="urn:a"><Child xmlns="urn:b" version="1.1"><Item Año="2025"/><Wrap><Item Año="2026"/></Wrap></Child></Root>`), "http://example.test/main")
	if err != nil || got != "20252026yes" {
		t.Fatalf("%q %v", got, err)
	}
}
