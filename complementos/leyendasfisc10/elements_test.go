package leyendasfisc10

import "testing"

func TestLeyendas(t *testing.T) {
	p := NewLeyendasFiscales(nil)
	p.MultiLeyenda(Attrs{"textoLeyenda": "A & B", "norma": "1", "disposicionFiscal": "Ley"}, Attrs{"textoLeyenda": "Segunda"})
	expected := `<leyendasFisc:LeyendasFiscales xmlns:leyendasFisc="http://www.sat.gob.mx/leyendasFiscales" xsi:schemaLocation="http://www.sat.gob.mx/leyendasFiscales http://www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd" version="1.0"><leyendasFisc:Leyenda disposicionFiscal="Ley" norma="1" textoLeyenda="A &amp; B"/><leyendasFisc:Leyenda textoLeyenda="Segunda"/></leyendasFisc:LeyendasFiscales>`
	if p.String() != expected {
		t.Fatal(p.String())
	}
}
