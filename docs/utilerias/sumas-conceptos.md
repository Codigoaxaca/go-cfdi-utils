# Sumas de conceptos (`sumasconceptos`)

El paquete `sumasconceptos` calcula los totales de un comprobante con base en sus conceptos:

- `SubTotal`: suma de los `Importe` de los conceptos.
- `Descuento`: suma de los `Descuento` de los conceptos.
- Traslados agrupados por `Impuesto`, `TipoFactor` y `TasaOCuota`, con su `Base` e `Importe`.
- Traslados exentos agrupados por `Impuesto` (solo `Base`).
- Retenciones agrupadas por `Impuesto`.
- Impuestos locales (complemento `implocal`) trasladados y retenidos.
- `Total = SubTotal - Descuento + Trasladados - Retenidos + LocalesTrasladados - LocalesRetenidos`.

Normalmente se usa a través de `cfdi40.Creator`:

```go
creator.AddSumasConceptos(nil, 2) // calcula y escribe con 2 decimales
```

`AddSumasConceptos` escribe:

- `SubTotal`, `Total` y `Descuento` en el comprobante (`Descuento` solo si algún concepto lo
  tiene o si es mayor a cero).
- El nodo `cfdi:Impuestos` con `TotalImpuestosTrasladados`, `TotalImpuestosRetenidos`, los
  traslados (con `Base`), los exentos y las retenciones. Si no hay impuestos el nodo se elimina.
- `TotaldeTraslados` y `TotaldeRetenciones` del complemento `implocal:ImpuestosLocales` si existe.

## Uso directo

También sirve para revisar un CFDI recibido:

```go
c, err := cfdi40.ReadFile("cfdi.xml")
if err != nil { /* ... */ }

sumas := sumasconceptos.New(c, 2)
fmt.Println(sumas.SubTotal, sumas.Descuento, sumas.Total)
for _, t := range sumas.Traslados {
	fmt.Println(t.Impuesto, t.TipoFactor, t.TasaOCuota, t.Base, t.Importe)
}
if iva, ok := sumas.Traslado("002", "Tasa", "0.160000"); ok {
	fmt.Println("IVA 16%:", iva.Importe)
}
```

O calcularlas desde el `Creator` y modificarlas antes de escribirlas:

```go
sumas := creator.BuildSumasConceptos(2)
// ... ajustes ...
creator.AddSumasConceptos(sumas, 2)
```

## Redondeo

Los importes se redondean "mitad hacia arriba" sobre la representación decimal del número, igual
que `round()` de PHP, de forma que `1.005` redondea a `1.01` (y no a `1.00` como ocurriría con un
redondeo binario ingenuo). Los importes agrupados se redondean a la precisión indicada antes de
sumarse al total, igual que la versión PHP.

La precisión normalmente es la cantidad de decimales de la moneda (2 para MXN, USD y EUR).
