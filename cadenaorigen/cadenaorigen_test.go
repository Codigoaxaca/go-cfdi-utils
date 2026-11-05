package cadenaorigen

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

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
	// FS overrides the embedded document, with HTTP/HTTPS sharing one key.
	override := fstest.MapFS{"www.sat.gob.mx/sitio_internet/cfd/4/cadenaoriginal_4_0/cadenaoriginal_4_0.xslt": {Data: []byte(styleHead + `<xsl:template match="/">override</xsl:template></xsl:stylesheet>`)}}
	got, err = NewBuilder(WithFS(override)).Build([]byte(`<x/>`), strings.Replace(XsltCfdi40, "http:", "https:", 1))
	if err != nil || got != "override" {
		t.Fatalf("%q %v", got, err)
	}
}
func TestHTTPCache(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		fmt.Fprint(w, styleHead+`<xsl:template match="/">downloaded</xsl:template></xsl:stylesheet>`)
	}))
	defer server.Close()
	b := NewBuilder(WithHTTPClient(server.Client()))
	for i := 0; i < 3; i++ {
		got, err := b.Build([]byte(`<x/>`), server.URL+"/main.xslt")
		if err != nil || got != "downloaded" {
			t.Fatalf("%q %v", got, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatal(hits.Load())
	}
	if _, err := NewBuilder().Build([]byte(`<x/>`), server.URL+"/main.xslt"); err == nil {
		t.Fatal("unexpected network fallback")
	}
}
func TestAllStylesheetsCompile(t *testing.T) {
	b := NewBuilder()
	err := fs.WalkDir(resources, "resources", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".xslt") {
			_, e := b.compile("http://" + strings.TrimPrefix(path, "resources/"))
			if e != nil {
				t.Errorf("%s: %v", path, e)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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
