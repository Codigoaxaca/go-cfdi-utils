package tfd11_test

import (
	"fmt"
	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/complementos/tfd11"
)

func Example() {
	comprobante := cfdi40.NewComprobante(nil)
	complemento := tfd11.NewTimbreFiscalDigital(nil)
	complemento.SetAttrs(tfd11.Attrs{"UUID": "00000000-1111-2222-3333-00000000000A", "FechaTimbrado": "2024-07-17T12:00:00"})
	comprobante.AddComplemento(complemento)
	fmt.Println(comprobante.SearchNode("cfdi:Complemento", complemento.Name()) != nil)
	// Output: true
}
