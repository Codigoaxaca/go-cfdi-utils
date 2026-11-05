# Complemento de pagos 2.0

El paquete `complementos/pagos20` contiene los elementos del *Complemento para recepción de
pagos 2.0* y una calculadora para los atributos que más problemas dan: `Totales`, los impuestos
de cada pago (`ImpuestosP`) y el monto mínimo del pago.

```go
import "github.com/codigoaxaca/go-cfdi-utils/complementos/pagos20"
```

## Elementos

| Elemento | Métodos |
|----------|---------|
| `Pagos` (raíz, `Version="2.0"`) | `Totales()`, `AddTotales(attrs)`, `AddPago(attrs) *Pago`, `MultiPago(attrs...)`, `WriteTotales(opts...)` |
| `Pago` | `AddDoctoRelacionado(attrs) *DoctoRelacionado`, `MultiDoctoRelacionado(attrs...)`, `ImpuestosP()`, `AddImpuestosP(attrs)` |
| `DoctoRelacionado` | `ImpuestosDR()`, `AddImpuestosDR(attrs)` |
| `ImpuestosDR` | `RetencionesDR()`, `AddRetencionesDR(attrs)`, `TrasladosDR()`, `AddTrasladosDR(attrs)` |
| `RetencionesDR` / `TrasladosDR` | `AddRetencionDR(attrs)`, `MultiRetencionDR(...)` / `AddTrasladoDR(attrs)`, `MultiTrasladoDR(...)` |
| `ImpuestosP` | `RetencionesP()`, `AddRetencionesP(attrs)`, `TrasladosP()`, `AddTrasladosP(attrs)` |
| `RetencionesP` / `TrasladosP` | `AddRetencionP(attrs)`, `MultiRetencionP(...)` / `AddTrasladoP(attrs)`, `MultiTrasladoP(...)` |

## Ejemplo completo: CFDI de pagos (REP)

```go
csd, err := credentials.OpenFiles("csd.cer", "csd.key", "contraseña")
if err != nil { log.Fatal(err) }

creator := cfdi40.NewCreator(cfdi40.Attrs{
	"Fecha":             "2023-06-13T14:15:16",
	"TipoDeComprobante": "P",
	"LugarExpedicion":   "52000",
	"Moneda":            "XXX",
	"Exportacion":       "01",
	"SubTotal":          "0",
	"Total":             "0",
})
creator.PutCertificado(csd.Certificate(), true)

comprobante := creator.Comprobante()
comprobante.AddEmisor(cfdi40.Attrs{"RegimenFiscal": "601"})
comprobante.AddReceptor(cfdi40.Attrs{
	"Rfc":                     "COSC8001137NA",
	"Nombre":                  "CARLOS CORTES SOTO",
	"RegimenFiscalReceptor":   "605",
	"UsoCFDI":                 "CP01",
	"DomicilioFiscalReceptor": "52000",
})
comprobante.AddConcepto(cfdi40.Attrs{
	"ClaveProdServ": "84111506",
	"Cantidad":      "1",
	"ClaveUnidad":   "ACT",
	"Descripcion":   "Pago",
	"ValorUnitario": "0",
	"Importe":       "0",
	"ObjetoImp":     "01",
})

pagos := pagos20.NewPagos(nil)
pago := pagos.AddPago(pagos20.Attrs{
	"FechaPago":    "2023-06-12T17:18:19",
	"FormaDePagoP": "03",
	"MonedaP":      "MXN",
	"TipoCambioP":  "1",
	"Monto":        "15000.00",
	"NumOperacion": "963852",
})
pago.AddDoctoRelacionado(pagos20.Attrs{
	"IdDocumento":      "00000000-1111-2222-3333-00000000000A",
	"MonedaDR":         "MXN",
	"EquivalenciaDR":   "1",
	"NumParcialidad":   "2",
	"ImpSaldoAnt":      "12000.00",
	"ImpPagado":        "12000.00",
	"ImpSaldoInsoluto": "0",
	"ObjetoImpDR":      "02",
}).AddImpuestosDR().AddTrasladosDR().AddTrasladoDR(pagos20.Attrs{
	"ImpuestoDR":   "002",
	"TipoFactorDR": "Tasa",
	"TasaOCuotaDR": "0.160000",
	"BaseDR":       "10344.83",
	"ImporteDR":    "1655.17",
})

// calcula y escribe Totales e ImpuestosP (y Monto si no se indicó)
if _, err := pagos.WriteTotales(); err != nil {
	log.Fatal(err)
}

comprobante.AddComplemento(pagos)
creator.MoveSatDefinitionsToComprobante()
if err := creator.AddSello(csd.PrivateKey()); err != nil {
	log.Fatal(err)
}
asserts := creator.Validate()
```

> Este ejemplo reproduce (con dos documentos relacionados) el archivo de pruebas
> `testdata/cfdi/created-cfdi40-pago20-valid.xml` creado con la versión PHP; la prueba
> `TestCreateComprobanteConPagos20` comprueba que el resultado, incluyendo el sello, es idéntico.

## Calculadora

La calculadora obtiene, a partir de los documentos relacionados:

- `Totales`: `MontoTotalPagos`, `TotalRetencionesIVA`, `TotalRetencionesISR`,
  `TotalRetencionesIEPS`, `TotalTrasladosBaseIVA16`, `TotalTrasladosImpuestoIVA16`, etc.
  (siempre en MXN).
- Los impuestos de cada pago (`ImpuestosP`) agrupados a partir de los `ImpuestosDR` convertidos
  con `EquivalenciaDR`.
- El monto mínimo de cada pago.

Usa aritmética decimal exacta (`math/big`, 24 decimales), igual que la versión PHP con BCMath.

```go
// calcular y escribir en un solo paso
result, err := pagos.WriteTotales()

// calcular con opciones y escribir después
calculator := pagos20.NewCalculator(
	pagos20.WithPaymentTaxesPrecision(2),                         // decimales de ImpuestosP (6 por omisión)
	pagos20.WithCurrencies(map[string]int{"MXN": 2, "USD": 2, "EUR": 2}), // decimales por moneda
	pagos20.WithTaxesTruncated(),                                // truncar en lugar de redondear
)
result, err := calculator.Calculate(pagos)
if err != nil { /* ... */ }
err = pagos20.Write(pagos, result)

// o con la función de paquete
result, err := pagos20.Calculate(pagos, pagos20.WithPaymentTaxesPrecision(2))
```

`Result` contiene:

- `Totales pagos20.Attrs`: los atributos de `Totales` que aplican.
- `Pagos []PaymentResult`: por cada pago, `Monto`, `MontoMinimo`, `TipoCambioP` e `Impuestos`
  (`[]TaxResult` con `Tipo`, `Impuesto`, `TipoFactor`, `TasaOCuota`, `Base` e `Importe`).

`Write` reemplaza `Totales` e `ImpuestosP` y establece `Monto` solo si el pago no lo tenía.
