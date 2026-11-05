package implocal10

import "testing"

func TestImpuestosLocales(t *testing.T) {
	p := NewImpuestosLocales(Attrs{"TotaldeRetenciones": "1.00", "TotaldeTraslados": "2.00"})
	p.AddTrasladoLocal(Attrs{"Importe": "2.00", "TasadeTraslado": "2", "ImpLocTrasladado": "ISH"})
	p.AddRetencionLocal(Attrs{"Importe": "1.00", "TasadeRetencion": "1", "ImpLocRetenido": "Local"})
	expected := `<implocal:ImpuestosLocales xmlns:implocal="http://www.sat.gob.mx/implocal" xsi:schemaLocation="http://www.sat.gob.mx/implocal http://www.sat.gob.mx/sitio_internet/cfd/implocal/implocal.xsd" version="1.0" TotaldeRetenciones="1.00" TotaldeTraslados="2.00"><implocal:RetencionesLocales ImpLocRetenido="Local" TasadeRetencion="1" Importe="1.00"/><implocal:TrasladosLocales ImpLocTrasladado="ISH" TasadeTraslado="2" Importe="2.00"/></implocal:ImpuestosLocales>`
	if p.String() != expected {
		t.Fatal(p.String())
	}
}
