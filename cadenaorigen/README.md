# cadenaorigen — notas de implementación

Documentación de uso: [docs/componentes/cadena-de-origen.md](../docs/componentes/cadena-de-origen.md).

- `Build(xml)` detecta CFDI 4.0, CFDI 3.3, TimbreFiscalDigital 1.1 y Retenciones 2.0.
  `BuildTFD(xml)` recibe un TFD 1.1 con su propio espacio de nombres declarado.
  `LocationFor(xml)` devuelve la URL oficial del XSLT. Los documentos no soportados devuelven
  un error que envuelve `ErrUnsupportedDocument`.
- Las hojas de estilo vienen del almacén `satresources` (por omisión `satresources.Default()`,
  o el de `WithStore`). `WithFS(fsys)` usa solo archivos propios `host/ruta`, sin descargas.
- `XsltBuilder` compila cada XSLT una vez por versión del almacén: cuando se activa una versión
  nueva, la siguiente cadena se genera con ella. Es seguro para uso concurrente y su valor cero
  funciona.

## Intérprete

El intérprete está en `internal/xslt`, porque `satresources` también lo usa para verificar una
versión nueva antes de activarla. Interpreta el subconjunto de XSLT que usa el SAT; no es un
procesador XSLT general.

- Instrucciones: `stylesheet`/`transform`, `output`, `include`, `template`, `param`,
  `apply-templates`, `for-each`, `call-template`, `with-param`, `if`, `value-of`, `text`.
- XPath: rutas de hijos, atributos y descendientes, comodín de elementos, variables,
  comparaciones de igualdad con literales y `normalize-space(string(...))`.
- `Compile` revisa de forma estática todas las plantillas: cualquier instrucción, atributo,
  patrón o expresión fuera del subconjunto es un error de compilación, no una cadena incorrecta.
- Los `include` se expanden en orden de documento; una URL repetida se carga una vez y gana la
  última plantilla que coincide.
- Los prefijos se resuelven en el ámbito de espacios de nombres de cada instrucción.
- `normalize-space` solo colapsa espacios XML (conserva espacios no separables).
- Un atributo opcional existente pero vacío es un conjunto de nodos verdadero y emite separador.
- Las reglas predeterminadas copian el texto de elementos sin plantilla, incluidos complementos
  desconocidos; así lo hace `xsltproc`. Las plantillas del SAT omiten la `Addenda`.
- No se agrega salto de línea final.

## Pruebas

Las pruebas usan el respaldo incluido (`satresources/embedded`) con un almacén sin conexión
(`internal/testresources`), así que no requieren red. `go generate ./satresources/embedded`
actualiza el respaldo desde el SAT.

`python3 cadenaorigen/testdata/generate_goldens.py` regenera los resultados esperados con
`/usr/bin/xsltproc` usando copias locales de los XSLT del respaldo. Las pruebas comparan todos
los CFDI de `testdata/`, Retenciones 2.0, los TFD extraídos de CFDI reales y casos construidos
(complementos, codificación, espacios). La referencia PHP `cfdi33-valid` usa el certificado
30001000000500003416 mientras que el XML de prueba tiene 30001000000400002434; la comparación
contra PHP ajusta solo una copia en memoria de ese atributo.
