package implocal10_test

import (
	"fmt"
	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/complementos/implocal10"
)

func Example() {
	comprobante := cfdi40.NewComprobante(nil)
	complemento := implocal10.NewImpuestosLocales(nil)
	complemento.AddTrasladoLocal(implocal10.Attrs{"ImpLocTrasladado": "ISH", "TasadeTraslado": "3", "Importe": "30.00"})
	comprobante.AddComplemento(complemento)
	fmt.Println(comprobante.SearchNode("cfdi:Complemento", complemento.Name()) != nil)
	// Output: true
}
