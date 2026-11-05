# Limpieza de un CFDI (`cleaner`)

Frecuentemente se reciben CFDI que fueron sellados y timbrados correctamente pero que después
fueron alterados con información que no pertenece a la cadena original. El SAT lo permite, por
ejemplo agregar una addenda.

El problema es que esa información suele tener errores:

- El nodo `cfdi:Addenda` contiene nodos sin espacio de nombres o sin XSD.
- Hay espacios de nombres declarados que no se usan o que no son del SAT.
- Los esquemas declarados no se pueden descargar o el contenido no los cumple.
- Hay varios nodos `cfdi:Complemento` (el Anexo 20 indica que solo puede haber uno).

Todo esto termina en errores de validación. El paquete `cleaner` corrige estos problemas sin
alterar la cadena original ni el sello.

```go
import "github.com/codigoaxaca/go-cfdi-utils/cleaner"

limpio, err := cleaner.Clean(sucio)
```

## Pasos de limpieza

1. Cambia la definición incorrecta `xmlns:schemaLocation` por `xsi:schemaLocation`.
2. Remueve la definición de CFDI 3 sin prefijo (`xmlns="http://www.sat.gob.mx/cfd/3"`) si existe la
   definición con prefijo `xmlns:cfdi`.
3. Remueve el nodo `cfdi:Addenda`.
4. Remueve de `xsi:schemaLocation` los espacios de nombres que no tienen a continuación una URL
   que termine en `.xsd`.
5. Remueve los nodos que no pertenecen a espacios de nombres del SAT (`http://www.sat.gob.mx/`).
6. Remueve de `xsi:schemaLocation` los pares que no pertenecen al SAT.
7. Remueve las declaraciones de espacios de nombres que no son del SAT y ya no se usan.
8. Colapsa los nodos `cfdi:Complemento` en uno solo, respetando el orden de aparición para que
   se genere exactamente la misma cadena original.
9. Corrige las URL de XSD conocidas del SAT escritas con otras mayúsculas.

Los pasos 1 y 2 trabajan sobre el texto (función `cleaner.BeforeLoad`), el resto sobre los nodos.

## Ejecutar pasos individuales

```go
c, err := cleaner.New(sucio) // ejecuta BeforeLoad e interpreta el XML
if err != nil { /* ... */ }
c.RemoveAddenda()
c.CollapseComprobanteComplemento()
limpio := c.Root().XML()
```

Métodos disponibles: `RemoveAddenda`, `RemoveIncompleteSchemaLocations`, `RemoveNonSatNSNodes`,
`RemoveNonSatNSSchemaLocations`, `RemoveUnusedNamespaces`, `CollapseComprobanteComplemento`,
`FixKnownSchemaLocationsXsdUrls` y `CleanAll` (todos).
