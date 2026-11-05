package xslt_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/internal/xslt"
	"github.com/codigoaxaca/go-cfdi-utils/satresources"
	"github.com/codigoaxaca/go-cfdi-utils/satresources/embedded"
)

func loader(files map[string]string) xslt.Loader {
	return func(loc string) ([]byte, error) {
		k, err := xslt.Key(loc)
		if err != nil {
			return nil, err
		}
		if s, ok := files[k]; ok {
			return []byte(s), nil
		}
		return nil, fs.ErrNotExist
	}
}

// Las hojas de estilo actuales del SAT pasan la revisión estricta.
func TestSATEntryPointsCompile(t *testing.T) {
	fsys := embedded.FS()
	load := func(loc string) ([]byte, error) {
		k, err := xslt.Key(loc)
		if err != nil {
			return nil, err
		}
		return fs.ReadFile(fsys, "files/"+k)
	}
	for _, loc := range []string{satresources.XsltCfdi40, satresources.XsltCfdi33, satresources.XsltTfd11, satresources.XsltRetenciones20} {
		if _, err := xslt.Compile(loc, load); err != nil {
			t.Errorf("%s: %v", loc, err)
		}
	}
}

const head = `<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:a="urn:a" version="2.0">`

// Cualquier construcción fuera del subconjunto se rechaza al compilar, para no
// generar en silencio una cadena incorrecta.
func TestCompileRejectsUnsupported(t *testing.T) {
	cases := map[string]string{
		"instrucción":           `<xsl:template match="/"><xsl:choose/></xsl:template>`,
		"predicado":             `<xsl:template match="/"><xsl:value-of select="./a:X[1]/@b"/></xsl:template>`,
		"función":               `<xsl:template match="/"><xsl:value-of select="concat(./@a, 'x')"/></xsl:template>`,
		"distinto":              `<xsl:template match="/"><xsl:if test="./@a != 'x'">y</xsl:if></xsl:template>`,
		"modo":                  `<xsl:template match="/"><xsl:apply-templates select="./*" mode="m"/></xsl:template>`,
		"orden":                 `<xsl:template match="/"><xsl:apply-templates select="./*"><xsl:sort select="@a"/></xsl:apply-templates></xsl:template>`,
		"patrón":                `<xsl:template match="a:X/a:Y">y</xsl:template>`,
		"prefijo":               `<xsl:template match="/"><xsl:value-of select="./b:X"/></xsl:template>`,
		"plantilla":             `<xsl:template match="/"><xsl:call-template name="nada"/></xsl:template>`,
		"literal":               `<xsl:template match="/"><b>x</b></xsl:template>`,
		"atributo comodín":      `<xsl:template match="/"><xsl:value-of select="./@*"/></xsl:template>`,
		"import":                `<xsl:import href="x.xslt"/>`,
		"variable":              `<xsl:variable name="v" select="1"/>`,
		"value-of sin select":   `<xsl:template match="/"><xsl:value-of/></xsl:template>`,
		"patrón comodín":        `<xsl:template match="*">X</xsl:template>`,
		"param con texto":       `<xsl:template match="/"><xsl:param name="p">foo</xsl:param><xsl:value-of select="$p"/></xsl:template>`,
		"value-of con texto":    `<xsl:template match="/"><xsl:value-of select="./@a">x</xsl:value-of></xsl:template>`,
		"with-param con texto":  `<xsl:template name="t"><xsl:param name="p"/></xsl:template><xsl:template match="/"><xsl:call-template name="t"><xsl:with-param name="p" select="./@a">x</xsl:with-param></xsl:call-template></xsl:template>`,
		"with-param sin select": `<xsl:template name="t"><xsl:param name="p"/></xsl:template><xsl:template match="/"><xsl:call-template name="t"><xsl:with-param name="p"/></xsl:call-template></xsl:template>`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := xslt.Compile("http://test.invalid/a.xslt", loader(map[string]string{"test.invalid/a.xslt": head + body + `</xsl:stylesheet>`}))
			if err == nil {
				t.Fatal("se esperaba un error")
			}
		})
	}
}

func TestCompileRejectsIncludeCycles(t *testing.T) {
	files := map[string]string{
		"test.invalid/a.xslt": head + `<xsl:include href="b.xslt"/><xsl:template match="/">a</xsl:template></xsl:stylesheet>`,
		"test.invalid/b.xslt": head + `<xsl:include href="a.xslt"/></xsl:stylesheet>`,
	}
	if _, err := xslt.Compile("http://test.invalid/a.xslt", loader(files)); err == nil || !strings.Contains(err.Error(), "circular") {
		t.Fatalf("se esperaba error por inclusión circular: %v", err)
	}
	// incluir el mismo archivo desde dos lugares no es un ciclo
	files["test.invalid/a.xslt"] = head + `<xsl:include href="c.xslt"/><xsl:include href="b.xslt"/><xsl:template match="/">a</xsl:template></xsl:stylesheet>`
	files["test.invalid/b.xslt"] = head + `<xsl:include href="c.xslt"/></xsl:stylesheet>`
	files["test.invalid/c.xslt"] = head + `<xsl:template name="c">c</xsl:template></xsl:stylesheet>`
	if _, err := xslt.Compile("http://test.invalid/a.xslt", loader(files)); err != nil {
		t.Fatal(err)
	}
}

func TestCompileAcceptsSubset(t *testing.T) {
	files := map[string]string{
		"test.invalid/main.xslt": head + `<xsl:include href="inc.xslt"/><xsl:output method="text"/>` +
			`<xsl:template match="/"><xsl:apply-templates select="/a:Root"/></xsl:template>` +
			`<xsl:template match="a:Root"><xsl:call-template name="R"><xsl:with-param name="v" select="./@Año"/></xsl:call-template>` +
			`<xsl:for-each select=".//a:Item"><xsl:if test="./@t='1'">|<xsl:value-of select="normalize-space(string(./@v))"/></xsl:if></xsl:for-each>` +
			`<xsl:text>|fin</xsl:text></xsl:template></xsl:stylesheet>`,
		"test.invalid/inc.xslt": head + `<xsl:template name="R"><xsl:param name="v"/>|<xsl:value-of select="$v"/></xsl:template></xsl:stylesheet>`,
	}
	s, err := xslt.Compile("http://test.invalid/main.xslt", loader(files))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := xslt.Parse([]byte(`<Root xmlns="urn:a" Año="2026"><Item t="1" v=" a  b "/><X><Item t="1" v="c"/></X><Item t="2" v="no"/></Root>`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.Transform(doc)
	if err != nil || out != "|2026|a b|c|fin" {
		t.Fatalf("%q %v", out, err)
	}
}

func TestKey(t *testing.T) {
	a, _ := xslt.Key("http://WWW.SAT.gob.mx/x/y.xsd")
	b, _ := xslt.Key("https://www.sat.gob.mx/x/y.xsd")
	if a != b || a != "www.sat.gob.mx/x/y.xsd" {
		t.Fatalf("%q %q", a, b)
	}
	for _, bad := range []string{"ftp://x/y", "http:///y", "http://x/y?z=1", "http://x/../y", "relativo.xsd"} {
		if _, err := xslt.Key(bad); err == nil || strings.TrimSpace(bad) == "" {
			t.Errorf("se aceptó %q", bad)
		}
	}
}
