package leyendasfisc10_test

import (
	"fmt"
	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/complementos/leyendasfisc10"
)

func Example() {
	comprobante := cfdi40.NewComprobante(nil)
	complemento := leyendasfisc10.NewLeyendasFiscales(nil)
	complemento.AddLeyenda(leyendasfisc10.Attrs{"disposicionFiscal": "Ley", "textoLeyenda": "Leyenda fiscal de ejemplo"})
	comprobante.AddComplemento(complemento)
	fmt.Println(comprobante.SearchNode("cfdi:Complemento", complemento.Name()) != nil)
	// Output: true
}
