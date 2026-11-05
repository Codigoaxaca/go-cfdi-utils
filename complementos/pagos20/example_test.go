package pagos20_test

import (
	"fmt"
	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/complementos/pagos20"
)

func Example() {
	comprobante := cfdi40.NewComprobante(nil)
	complemento := pagos20.NewPagos(nil)
	complemento.AddPago(pagos20.Attrs{"MonedaP": "MXN", "TipoCambioP": "1"}).AddDoctoRelacionado(pagos20.Attrs{"IdDocumento": "00000000-1111-2222-3333-00000000000A", "EquivalenciaDR": "1", "ImpPagado": "100.00"})
	if _, err := complemento.WriteTotales(); err != nil {
		panic(err)
	}
	comprobante.AddComplemento(complemento)
	fmt.Println(comprobante.SearchNode("cfdi:Complemento", complemento.Name()) != nil)
	// Output: true
}
