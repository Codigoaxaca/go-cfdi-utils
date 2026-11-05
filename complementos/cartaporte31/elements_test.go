package cartaporte31

import (
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"testing"
)

func TestCartaPorte(t *testing.T) {
	p := NewCartaPorte(Attrs{"IdCCP": "CCC123", "TranspInternac": "No"})
	p.FiguraTransporte().AddTiposFigura(Attrs{"TipoFigura": "01"}).AddDomicilio(Attrs{"Pais": "MEX", "Estado": "OAX", "CodigoPostal": "68000"})
	merc := p.AddMercancias(Attrs{"NumTotalMercancias": "1", "UnidadPeso": "KGM", "PesoBrutoTotal": "100"})
	auto := merc.Autotransporte()
	auto.Remolques().AddRemolque(Attrs{"Placa": "ABC123", "SubTipoRem": "CTR001"})
	auto.AddSeguros(Attrs{"PolizaRespCivil": "123", "AseguraRespCivil": "Seguro"})
	auto.AddIdentificacionVehicular(Attrs{"ConfigVehicular": "C2"})
	merc.AddMercancia(Attrs{"Descripcion": "Carga", "BienesTransp": "10101500", "Cantidad": "1", "ClaveUnidad": "H87", "PesoEnKg": "100"})
	p.Ubicaciones().AddUbicacion(Attrs{"TipoUbicacion": "Origen"})
	p.RegimenesAduaneros().AddRegimenAduaneroCCP(Attrs{"RegimenAduanero": "IMD"})
	expected := `<cartaporte31:CartaPorte xmlns:cartaporte31="http://www.sat.gob.mx/CartaPorte31" xsi:schemaLocation="http://www.sat.gob.mx/CartaPorte31 http://www.sat.gob.mx/sitio_internet/cfd/CartaPorte/CartaPorte31.xsd" Version="3.1" IdCCP="CCC123" TranspInternac="No"><cartaporte31:RegimenesAduaneros><cartaporte31:RegimenAduaneroCCP RegimenAduanero="IMD"/></cartaporte31:RegimenesAduaneros><cartaporte31:Ubicaciones><cartaporte31:Ubicacion TipoUbicacion="Origen"/></cartaporte31:Ubicaciones><cartaporte31:Mercancias PesoBrutoTotal="100" UnidadPeso="KGM" NumTotalMercancias="1"><cartaporte31:Mercancia BienesTransp="10101500" Descripcion="Carga" Cantidad="1" ClaveUnidad="H87" PesoEnKg="100"/><cartaporte31:Autotransporte><cartaporte31:IdentificacionVehicular ConfigVehicular="C2"/><cartaporte31:Seguros AseguraRespCivil="Seguro" PolizaRespCivil="123"/><cartaporte31:Remolques><cartaporte31:Remolque SubTipoRem="CTR001" Placa="ABC123"/></cartaporte31:Remolques></cartaporte31:Autotransporte></cartaporte31:Mercancias><cartaporte31:FiguraTransporte><cartaporte31:TiposFigura TipoFigura="01"><cartaporte31:Domicilio Estado="OAX" Pais="MEX" CodigoPostal="68000"/></cartaporte31:TiposFigura></cartaporte31:FiguraTransporte></cartaporte31:CartaPorte>`
	if p.String() != expected {
		t.Fatal(p.String())
	}
	if p.Mercancias().Node != merc.Node || p.AddMercancias(Attrs{"PesoNetoTotal": "90"}).Node != merc.Node {
		t.Fatal("hijo único duplicado")
	}
	if _, e := nodes.ParseXMLString(p.XML()); e != nil {
		t.Fatal(e)
	}
}
func TestTransportesYMercancias(t *testing.T) {
	p := NewCartaPorte(nil)
	merc := p.Mercancias()
	sea := merc.TransporteMaritimo().AddContenedor(Attrs{"TipoContenedor": "CM001"})
	sea.RemolquesCCP().MultiRemolqueCCP(Attrs{"PlacaCCP": "A"}, Attrs{"PlacaCCP": "B"})
	rail := merc.TransporteFerroviario()
	rail.MultiDerechosDePaso(nil, nil)
	rail.AddCarro(nil).MultiContenedor(Attrs{"TipoContenedor": "TC01"}, Attrs{"TipoContenedor": "TC02"})
	merc.AddTransporteAereo(nil)
	m := merc.AddMercancia(nil)
	m.MultiDocumentacionAduanera(nil, nil).MultiGuiasIdentificacion(nil, nil).MultiCantidadTransporta(nil, nil).AddDetalleMercancia(nil)
	if m.Len() != 7 || sea.RemolquesCCP().Len() != 2 {
		t.Fatal(p.String())
	}
	if rail.SearchNode("cartaporte31:Carro", "cartaporte31:Contenedor") == nil {
		t.Fatal("falta contenedor ferroviario")
	}
}
