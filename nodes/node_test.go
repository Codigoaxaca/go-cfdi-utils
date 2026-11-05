package nodes

import (
	"strings"
	"testing"
)

func TestNodeAttributes(t *testing.T) {
	n := New("root", Attrs{"b": "2", "a": "1"})
	if n.Attr("a") != "1" || n.Attr("b") != "2" {
		t.Fatalf("unexpected attrs %v", n.Attributes())
	}
	if got := n.Attributes(); got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("map attributes must be sorted, got %v", got)
	}
	if n.Attr("no-existe") != "" || n.HasAttr("no-existe") {
		t.Fatal("missing attribute must be empty")
	}
	n.SetAttr("vacio", "")
	if !n.HasAttr("vacio") {
		t.Fatal("empty attribute must exist")
	}
	n.RemoveAttr("a")
	if n.HasAttr("a") || len(n.Attributes()) != 2 {
		t.Fatal("attribute was not removed")
	}
}

func TestNodeAttributesOrder(t *testing.T) {
	n := New("cfdi:Emisor", nil)
	n.SetAttributesOrder("Rfc", "Nombre", "RegimenFiscal")
	n.SetAttrs(Attrs{"RegimenFiscal": "601", "Zeta": "z", "Nombre": "ACME", "Rfc": "AAA010101AAA", "xmlns:cfdi": "x"})
	var names []string
	for _, a := range n.Attributes() {
		names = append(names, a.Name)
	}
	if got := strings.Join(names, ","); got != "xmlns:cfdi,Rfc,Nombre,RegimenFiscal,Zeta" {
		t.Fatalf("unexpected order %s", got)
	}
}

func TestChildrenOrder(t *testing.T) {
	root := New("root", nil)
	root.SetChildrenOrder("a", "b", "c")
	root.AddChild(New("c", nil))
	root.AddChild(New("x", nil))
	root.AddChild(New("a", nil))
	root.AddChild(New("b", nil))
	root.AddChild(New("a", Attrs{"second": "1"}))
	var names []string
	for _, c := range root.Children() {
		names = append(names, c.Name())
	}
	if got := strings.Join(names, ","); got != "a,a,b,c,x" {
		t.Fatalf("unexpected order %s", got)
	}
}

func TestSearch(t *testing.T) {
	xml := `<r:root xmlns:r="urn:r"><r:a id="1"><r:b id="b1"/><r:b id="b2"/></r:a><r:a id="2"/></r:root>`
	root, err := ParseXMLString(xml)
	if err != nil {
		t.Fatal(err)
	}
	if root.SearchAttribute("r:a", "id") != "1" {
		t.Fatal("SearchAttribute failed")
	}
	if len(root.SearchNodes("r:a", "r:b")) != 2 {
		t.Fatal("SearchNodes failed")
	}
	if root.SearchNode("r:a", "r:c") != nil {
		t.Fatal("SearchNode must return nil")
	}
	if root.SearchAttribute("r:x", "id") != "" {
		t.Fatal("SearchAttribute must return empty")
	}
}

func TestParseAndMarshal(t *testing.T) {
	source := `<?xml version="1.0" encoding="UTF-8"?>
<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" Version="4.0" Nombre="A &amp; B &quot;C&quot; &lt;D&gt;">
  <!-- comentario -->
  <cfdi:Emisor Rfc="AAA010101AAA"/>
  <texto>hola &amp; adiós</texto>
</cfdi:Comprobante>`
	root, err := ParseXMLString(source)
	if err != nil {
		t.Fatal(err)
	}
	if root.Name() != "cfdi:Comprobante" || root.LocalName() != "Comprobante" || root.Prefix() != "cfdi" {
		t.Fatal("bad name")
	}
	if root.Attr("Nombre") != `A & B "C" <D>` {
		t.Fatalf("bad entity decoding: %q", root.Attr("Nombre"))
	}
	if root.Attr("xmlns:cfdi") != "http://www.sat.gob.mx/cfd/4" {
		t.Fatal("namespace declaration not preserved")
	}
	if v := root.FirstChild("texto").Value(); v != "hola & adiós" {
		t.Fatalf("bad value %q", v)
	}
	if root.Value() != "" {
		t.Fatal("whitespace must not be a value")
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" Version="4.0" Nombre="A &amp; B &quot;C&quot; &lt;D&gt;">
  <cfdi:Emisor Rfc="AAA010101AAA"/>
  <texto>hola &amp; adiós</texto>
</cfdi:Comprobante>
`
	if got := root.XML(); got != expected {
		t.Fatalf("unexpected xml:\n%s", got)
	}
	again, err := ParseXMLString(root.String())
	if err != nil {
		t.Fatal(err)
	}
	if again.String() != root.String() {
		t.Fatal("round trip failed")
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{"", "<a>", "<a></b>", "<a/><b/>", "texto"} {
		if _, err := ParseXMLString(src); err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestParseLatin1(t *testing.T) {
	src := []byte("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><a b=\"a\xf1o\"/>")
	n, err := ParseXML(src)
	if err != nil {
		t.Fatal(err)
	}
	if n.Attr("b") != "año" {
		t.Fatalf("bad latin1 decoding %q", n.Attr("b"))
	}
}

func TestMoveSatDefinitionsToRoot(t *testing.T) {
	src := `<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.sat.gob.mx/cfd/4 http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd">
<cfdi:Complemento>
<leyendasFisc:LeyendasFiscales xmlns:leyendasFisc="http://www.sat.gob.mx/leyendasFiscales" xsi:schemaLocation="http://www.sat.gob.mx/leyendasFiscales http://www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd" version="1.0"/>
<otro:Nodo xmlns:otro="http://example.com/otro" xsi:schemaLocation="http://example.com/otro http://example.com/otro.xsd"/>
</cfdi:Complemento>
</cfdi:Comprobante>`
	root, err := ParseXMLString(src)
	if err != nil {
		t.Fatal(err)
	}
	MoveSatDefinitionsToRoot(root)
	if root.Attr("xmlns:leyendasFisc") != "http://www.sat.gob.mx/leyendasFiscales" {
		t.Fatal("namespace not moved")
	}
	want := "http://www.sat.gob.mx/cfd/4 http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd http://www.sat.gob.mx/leyendasFiscales http://www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd"
	if root.Attr("xsi:schemaLocation") != want {
		t.Fatalf("bad schema location %q", root.Attr("xsi:schemaLocation"))
	}
	leyendas := root.SearchNode("cfdi:Complemento", "leyendasFisc:LeyendasFiscales")
	if leyendas.HasAttr("xmlns:leyendasFisc") || leyendas.HasAttr("xsi:schemaLocation") {
		t.Fatal("definitions not removed from child")
	}
	otro := root.SearchNode("cfdi:Complemento", "otro:Nodo")
	if !otro.HasAttr("xmlns:otro") || !otro.HasAttr("xsi:schemaLocation") {
		t.Fatal("non SAT definitions must stay")
	}
}

func TestClone(t *testing.T) {
	a := New("a", Attrs{"x": "1"}, New("b", nil))
	b := a.Clone()
	b.SetAttr("x", "2")
	b.Child(0).SetAttr("y", "1")
	if a.Attr("x") != "1" || a.Child(0).HasAttr("y") {
		t.Fatal("clone is not deep")
	}
}
