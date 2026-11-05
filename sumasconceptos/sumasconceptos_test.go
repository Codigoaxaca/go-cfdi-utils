package sumasconceptos

import (
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

func TestSumasFromXML(t *testing.T) {
	root, err := nodes.ParseXMLFile("../testdata/cfdi/created-with-discounts-40.xml")
	if err != nil {
		t.Fatal(err)
	}
	s := New(root, 2)
	if s.SubTotal != 3300 || s.Descuento != 500 || s.Total != 3248 || s.ImpuestosTrasladados != 448 {
		t.Fatalf("sumas inesperadas %+v", s)
	}
	tr, ok := s.Traslado("002", "Tasa", "0.160000")
	if !ok || tr.Base != 2800 || tr.Importe != 448 {
		t.Fatalf("traslado inesperado %+v", tr)
	}
	if !s.FoundAnyConceptWithDiscount || s.HasRetenciones() || s.HasExentos() {
		t.Fatal("banderas inesperadas")
	}
}

func TestSumasGroupsRetencionesExentosLocales(t *testing.T) {
	xml := `<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" xmlns:implocal="http://www.sat.gob.mx/implocal">
<cfdi:Conceptos>
  <cfdi:Concepto Importe="100.005">
    <cfdi:Impuestos>
      <cfdi:Traslados>
        <cfdi:Traslado Base="100" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="16.004"/>
        <cfdi:Traslado Base="100" Impuesto="003" TipoFactor="Exento"/>
      </cfdi:Traslados>
      <cfdi:Retenciones>
        <cfdi:Retencion Base="100" Impuesto="001" TipoFactor="Tasa" TasaOCuota="0.100000" Importe="10"/>
        <cfdi:Retencion Base="100" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.106667" Importe="10.6667"/>
      </cfdi:Retenciones>
    </cfdi:Impuestos>
  </cfdi:Concepto>
  <cfdi:Concepto Importe="50">
    <cfdi:Impuestos>
      <cfdi:Traslados>
        <cfdi:Traslado Base="50" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="8.001"/>
        <cfdi:Traslado Base="50" Impuesto="003" TipoFactor="Exento"/>
      </cfdi:Traslados>
      <cfdi:Retenciones>
        <cfdi:Retencion Base="50" Impuesto="001" TipoFactor="Tasa" TasaOCuota="0.100000" Importe="5"/>
      </cfdi:Retenciones>
    </cfdi:Impuestos>
  </cfdi:Concepto>
</cfdi:Conceptos>
<cfdi:Complemento>
  <implocal:ImpuestosLocales>
    <implocal:TrasladosLocales ImpLocTrasladado="ISH" TasadeTraslado="3.00" Importe="4.50"/>
    <implocal:RetencionesLocales ImpLocRetenido="CEDULAR" TasadeRetencion="1.00" Importe="1.50"/>
  </implocal:ImpuestosLocales>
</cfdi:Complemento>
</cfdi:Comprobante>`
	root, err := nodes.ParseXMLString(xml)
	if err != nil {
		t.Fatal(err)
	}
	s := New(root, 2)
	if s.SubTotal != 150.01 {
		t.Fatalf("subtotal %v", s.SubTotal)
	}
	if len(s.Traslados) != 1 || s.Traslados[0].Importe != 24.01 || s.Traslados[0].Base != 150 {
		t.Fatalf("traslados %+v", s.Traslados)
	}
	if len(s.Exentos) != 1 || s.Exentos[0].Base != 150 {
		t.Fatalf("exentos %+v", s.Exentos)
	}
	if len(s.Retenciones) != 2 || s.Retenciones[0].Importe != 15 || s.Retenciones[1].Importe != 10.67 || s.ImpuestosRetenidos != 25.67 {
		t.Fatalf("retenciones %+v", s.Retenciones)
	}
	if s.LocalesImpuestosTrasladados != 4.5 || s.LocalesImpuestosRetenidos != 1.5 {
		t.Fatalf("locales %+v", s)
	}
	// 150.01 + 24.01 - 25.67 + 4.50 - 1.50
	if s.Total != 151.35 {
		t.Fatalf("total %v", s.Total)
	}
}
