# Validaciones de CFDI 4.0

Estas validaciones se ejecutan tanto en `creator.Validate()` (CFDI creados) como en
`cfdi40.NewValidator().ValidateXML(...)` (CFDI recibidos), en este orden. Todas están en el
paquete `validate/cfdi40validators`.

Las primeras cinco son las mismas que tiene la versión PHP para CFDI 4.0. Las demás
(*validaciones estándar*) la versión PHP solo las tiene para CFDI 3.3; aquí se adaptaron a las
reglas de CFDI 4.0.

Estados: `OK` se cumple, `ERROR` no se cumple, `NONE` no aplica o no se pudo comprobar (con
una explicación).

## XmlFollowSchema

Valida el XML contra los esquemas XSD declarados en `xsi:schemaLocation` (ver
[esquemas XSD](esquemas-xsd.md)). Si falla, se detiene el resto de validaciones.

- `XSD01`: El contenido XML sigue los esquemas XSD

## XmlDefinition

- `XML01`: El XML implementa el namespace `http://www.sat.gob.mx/cfd/4` con el prefijo cfdi
- `XML02`: El nodo principal se llama cfdi:Comprobante
- `XML03`: La versión es 4.0

## SelloDigitalCertificado

Valida el sello del comprobante y el certificado incluido en él.

- `SELLO01`: Se puede obtener el certificado del comprobante
- `SELLO02`: El número de certificado del comprobante igual al encontrado en el certificado
- `SELLO03`: El RFC del comprobante igual al encontrado en el certificado
- `SELLO04`: El nombre del emisor del comprobante es igual al encontrado en el certificado.
  Para personas morales se compara sin el régimen de capital (como lo exige CFDI 4.0); un
  nombre vacío es error.
- `SELLO05`: La fecha del documento es mayor o igual a la fecha de inicio de vigencia del certificado
- `SELLO06`: La fecha del documento menor o igual a la fecha de fin de vigencia del certificado
- `SELLO07`: El sello del comprobante está en base 64
- `SELLO08`: El sello del comprobante coincide con el certificado y la cadena de origen generada

`SELLO03` y `SELLO04` no se evalúan en CFDI de registro fiscal (`registrofiscal:CFDIRegistroFiscal`)
emitidos con el certificado del SAT. La fecha del comprobante se interpreta en hora del centro
de México (UTC-6).

## TimbreFiscalDigitalSello

Posiblemente la **validación más importante** de un CFDI recibido: comprueba que el CFDI no fue
modificado después de timbrarse.

- `TFDSELLO01`: El Sello SAT del Timbre Fiscal Digital corresponde al certificado SAT

Proceso:

1. Obtiene el `tfd:TimbreFiscalDigital`; si no existe no hay qué validar (`NONE`).
2. Verifica que sea versión 1.1.
3. Verifica que `SelloCFD` sea igual al `Sello` del comprobante.
4. Verifica que `NoCertificadoSAT` sea un número válido (20 dígitos).
5. Obtiene el certificado del SAT de `https://rdc.sat.gob.mx/` (con caché).
6. Genera la cadena original del timbre.
7. Verifica que `SelloSAT` corresponda a la cadena original con el certificado del SAT.

Si alguien altera el contenido del comprobante sin tocar el timbre ni el `Sello`, esta validación
pasa, pero `SELLO08` falla porque el sello ya no coincide con la cadena original.

## TimbreFiscalDigitalVersion

- `TFDVERSION01`: Si existe el complemento timbre fiscal digital, entonces su versión debe ser 1.1

## ComprobanteDecimalesMoneda

Los importes no tienen más decimales de los que permite la moneda (MXN, USD y EUR: 2; XXX: 0).
Con otras monedas todos quedan en `NONE`.

- `MONDEC01`: El subtotal del comprobante no contiene más de los decimales de la moneda
- `MONDEC02`: El descuento del comprobante no contiene más de los decimales de la moneda
- `MONDEC03`: El total del comprobante no contiene más de los decimales de la moneda
- `MONDEC04`: El total de impuestos trasladados no contiene más de los decimales de la moneda
- `MONDEC05`: El total de impuestos retenidos no contiene más de los decimales de la moneda

## ComprobanteDescuento

- `DESCUENTO01`: Si existe el atributo descuento, entonces debe ser menor o igual que el subtotal y mayor o igual que cero

## ComprobanteFormaPago

- `FORMAPAGO01`: El campo forma de pago no debe existir cuando existe el complemento para recepción de pagos (`pago20:Pagos`)

## ComprobanteImpuestos

- `COMPIMPUESTOSC01`: Si existe el nodo impuestos entonces debe incluir el total de traslados y/o el total de retenciones, salvo que solo contenga traslados exentos
- `COMPIMPUESTOSC02`: Si existe al menos un traslado que no es exento entonces debe existir el total de traslados
- `COMPIMPUESTOSC03`: Si existe al menos una retención entonces debe existir el total de retenciones
- `COMPIMPUESTOSC04`: Si todos los traslados son exentos entonces no debe existir el total de traslados *(nueva en 4.0)*

## ComprobanteTipoCambio

- `TIPOCAMBIO01`: La moneda exista y no tenga un valor vacío
- `TIPOCAMBIO02`: Si la moneda es "MXN", entonces el tipo de cambio debe tener el valor "1" o no debe existir
- `TIPOCAMBIO03`: Si la moneda es "XXX", entonces el tipo de cambio no debe existir
- `TIPOCAMBIO04`: Si la moneda no es "MXN" ni "XXX", entonces el tipo de cambio debe seguir el patrón `[0-9]{1,18}(.[0-9]{1,6})?`

## ComprobanteTipoDeComprobante

- `TIPOCOMP01`: Si el tipo de comprobante es T, P ó N, entonces no debe existir las condiciones de pago
- `TIPOCOMP02`: Si el tipo de comprobante es T, P ó N, entonces no debe existir la definición de impuestos
- `TIPOCOMP03`: Si el tipo de comprobante es T ó P, entonces no debe existir la forma de pago
- `TIPOCOMP04`: Si el tipo de comprobante es T ó P, entonces no debe existir el método de pago
- `TIPOCOMP05`: Si el tipo de comprobante es T ó P, entonces no debe existir el descuento del comprobante
- `TIPOCOMP06`: Si el tipo de comprobante es T ó P, entonces no debe existir el descuento de los conceptos
- `TIPOCOMP07`: Si el tipo de comprobante es T ó P, entonces el subtotal debe ser cero
- `TIPOCOMP08`: Si el tipo de comprobante es T ó P entonces el total debe ser cero
- `TIPOCOMP09`: Si el tipo de comprobante es I, E ó N, entonces el valor unitario de todos los conceptos debe ser mayor que cero
- `TIPOCOMP10`: Si el tipo de comprobante es N entonces, la moneda debe ser MXN

## ComprobanteTotal

- `TOTAL01`: El atributo Total existe, no está vacío y cumple con el patrón `[0-9]+(.[0-9]+)?`

## ConceptoDescuento

- `CONCEPDESC01`: Si existe el atributo descuento en el concepto, entonces debe ser menor o igual que el importe y mayor o igual que cero

## ConceptoImpuestos

- `CONCEPIMPC01`: El nodo impuestos de un concepto debe incluir traslados y/o retenciones
- `CONCEPIMPC02`: Los traslados de los impuestos de un concepto deben tener una base y ser mayor a cero
- `CONCEPIMPC03`: No se debe registrar la tasa o cuota ni el importe cuando el tipo de factor de traslado es exento
- `CONCEPIMPC04`: Se debe registrar la tasa o cuota y el importe cuando el tipo de factor de traslado es tasa o cuota
- `CONCEPIMPC05`: Las retenciones de los impuestos de un concepto deben tener una base y ser mayor a cero
- `CONCEPIMPC06`: Las retenciones de los impuestos de un concepto deben tener un tipo de factor diferente de exento
- `CONCEPIMPC07`: Si el concepto es objeto de impuesto con desglose (ObjetoImp 02) entonces debe existir el nodo impuestos del concepto *(nueva en 4.0)*
- `CONCEPIMPC08`: Si el concepto no es objeto de impuesto, no está obligado al desglose o no causa impuesto (ObjetoImp 01, 03 o 04) entonces no debe existir el nodo impuestos del concepto *(nueva en 4.0)*

## EmisorRegimenFiscal

- `REGFIS01`: El régimen fiscal contenga un valor apropiado según el tipo de RFC emisor

Catálogo `c_RegimenFiscal` usado:

- Personas morales (RFC de 12 caracteres): 601, 603, 610, 620, 622, 623, 624, 626, 628.
- Personas físicas (RFC de 13 caracteres): 605, 606, 607, 608, 610, 611, 612, 614, 615, 616, 621, 625, 626, 629, 630.

## FechaComprobante

- `FECHA01`: La fecha del comprobante cumple con el formato
- `FECHA02`: La fecha existe en el comprobante y es mayor que 2022-01-01 y menor que el futuro

La fecha máxima es el momento de la validación más una tolerancia de 300 segundos. Se puede
configurar armando un `validate.MultiValidator` propio:

```go
cfdi40validators.FechaComprobante{Tolerance: 10 * time.Minute}
cfdi40validators.FechaComprobante{MaximumDate: time.Date(2024, 12, 31, 23, 59, 59, 0, cfdi40validators.MexicoLocation)}
```

## ReceptorResidenciaFiscal

- `RESFISC01`: Si el RFC no es XEXX010101000, entonces la residencia fiscal no debe existir
- `RESFISC02`: Si el RFC sí es XEXX010101000 y existe el complemento de comercio exterior, entonces la residencia fiscal debe establecerse y no puede ser "MEX"
- `RESFISC03`: Si el RFC sí es XEXX010101000 y se registró el número de registro de identificación fiscal, entonces la residencia fiscal debe establecerse y no puede ser "MEX"

## SumasConceptosComprobanteImpuestos

Calcula las sumas de los conceptos (con [`sumasconceptos`](../utilerias/sumas-conceptos.md)) y
las compara contra el comprobante y su nodo de impuestos, a la precisión de la moneda.

- `SUMAS01`: La suma de los importes de conceptos es igual al subtotal del comprobante
- `SUMAS02`: La suma de los descuentos es igual al descuento del comprobante
- `SUMAS03`: El cálculo del total es igual al total del comprobante
- `SUMAS04`: El cálculo de impuestos trasladados es igual al total de impuestos trasladados
- `SUMAS05`: Todos los impuestos trasladados existen en el comprobante
- `SUMAS06`: Todos los valores de los impuestos trasladados coinciden con el comprobante
- `SUMAS07`: No existen más nodos de impuestos trasladados en el comprobante de los que se han calculado
- `SUMAS08`: El cálculo de impuestos retenidos es igual al total de impuestos retenidos
- `SUMAS09`: Todos los impuestos retenidos existen en el comprobante
- `SUMAS10`: Todos los valores de los impuestos retenidos coinciden con el comprobante
- `SUMAS11`: No existen más nodos de impuestos retenidos en el comprobante de los que se han calculado
- `SUMAS12`: El cálculo del descuento debe ser menor o igual al cálculo del subtotal
- `SUMAS13`: Todas las bases de los impuestos trasladados coinciden con el comprobante *(nueva en 4.0)*

Además se agregan detalles por grupo de impuesto con códigos como `SUMAS06:001` o `SUMAS10:001`.

## Diferencias con las validaciones 3.3 de PHP

- Se reconocen los traslados exentos, que en 4.0 se escriben en el nodo de impuestos del
  comprobante con su `Base` (`COMPIMPUESTOSC01`/`02`/`04`, `SUMAS05`–`07`).
- La precisión de `SUMAS` es la de la moneda (2 si no se conoce).
- `FORMAPAGO01` busca el complemento `pago20:Pagos` (también acepta `pago10:Pagos`).
- `RESFISC02` reconoce Comercio Exterior 2.0 (`cce20`) además de 1.1.
- `REGFIS01` usa el catálogo de regímenes vigente para 4.0 (sin 609, con 626, 628, 629 y 630).
- `FECHA02` exige fechas a partir de 2022-01-01.
- Nuevas: `COMPIMPUESTOSC04`, `CONCEPIMPC07`, `CONCEPIMPC08`, `SUMAS13`.
- No se portaron `EMISORRFC01` ni `RECRFC01` (validación de la estructura del RFC).
