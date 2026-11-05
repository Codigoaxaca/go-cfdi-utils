package cartaporte31_test

import (
	"fmt"
	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/complementos/cartaporte31"
)

func Example() {
	comprobante := cfdi40.NewComprobante(nil)
	complemento := cartaporte31.NewCartaPorte(nil)
	complemento.Ubicaciones().AddUbicacion(cartaporte31.Attrs{"TipoUbicacion": "Origen", "RFCRemitenteDestinatario": "EKU9003173C9"}).AddDomicilio(cartaporte31.Attrs{"Estado": "OAX", "Pais": "MEX", "CodigoPostal": "68000"})
	complemento.Mercancias().AddMercancia(cartaporte31.Attrs{"BienesTransp": "10101500", "Descripcion": "Mercancía", "Cantidad": "1", "ClaveUnidad": "H87", "PesoEnKg": "100"})
	comprobante.AddComplemento(complemento)
	fmt.Println(comprobante.SearchNode("cfdi:Complemento", complemento.Name()) != nil)
	// Output: true
}
