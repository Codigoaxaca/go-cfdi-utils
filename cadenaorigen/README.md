# cadenaorigen — notas de implementación

Documentación de uso: [docs/componentes/cadena-de-origen.md](../docs/componentes/cadena-de-origen.md).

- `Build(xml)` detecta CFDI 4.0, CFDI 3.3, TimbreFiscalDigital 1.1 y Retenciones 2.0.
  `BuildTFD(xml)` recibe un TFD 1.1 con su propio espacio de nombres declarado.
  `LocationFor(xml)` devuelve la URL oficial del XSLT. Los documentos no soportados devuelven
  un error que envuelve `ErrUnsupportedDocument`.
- `NewBuilder(WithFS(fsys), WithHTTPClient(client))` permite recursos propios. `WithFS` resuelve
  `host/ruta` antes que los recursos embebidos; HTTP está desactivado salvo que se indique un
  cliente. Las URL http y https comparten la misma llave. Un builder se puede usar de forma
  concurrente y su valor cero también funciona. Los recursos compilados se conservan durante la
  vida del builder; crea otro para refrescarlos.

## Recursos

Se embeben 54 archivos XSLT: los cuatro puntos de entrada y sus `xsl:include` recursivos
(utilerías, complementos de CFDI como Pagos 1.0/2.0, Carta Porte 2.0/3.0/3.1, Nómina, Comercio
Exterior, etc., y complementos de Retenciones). Para actualizarlos ejecuta
`scripts/update-xslt.sh` (requiere Python 3).

## Subconjunto de XSLT soportado

Es un intérprete del subconjunto que usa el SAT, no un procesador XSLT general.

- Instrucciones: `stylesheet`/`transform`, `output`, `include`, `template`, `param`,
  `apply-templates`, `for-each`, `call-template`, `with-param`, `if`, `value-of`, `text`.
- XPath: rutas de hijos, atributos y descendientes, comodines, variables, comparaciones de
  atributos y `normalize-space(string(...))`.
- Los `include` se expanden en orden de documento; una URL repetida se carga una vez y gana la
  última plantilla que coincide.
- Los prefijos se resuelven en el ámbito de espacios de nombres de cada instrucción.
- `normalize-space` solo colapsa espacios XML (conserva espacios no separables).
- Un atributo opcional existente pero vacío es un conjunto de nodos verdadero y emite separador.
- Las reglas predeterminadas copian el texto de elementos sin plantilla, incluidos complementos
  desconocidos; así lo hace `xsltproc`. Las plantillas del SAT omiten la `Addenda`.
- No se agrega salto de línea final.

## Pruebas

`python3 cadenaorigen/testdata/generate_goldens.py` regenera los resultados esperados con
`/usr/bin/xsltproc` usando copias locales de los XSLT. Las pruebas comparan todos los CFDI de
`testdata/`, Retenciones 2.0, los TFD extraídos de CFDI reales y casos construidos (complementos,
codificación, espacios). La referencia PHP `cfdi33-valid` usa el certificado
30001000000500003416 mientras que el XML de prueba tiene 30001000000400002434; la comparación
contra PHP ajusta solo una copia en memoria de ese atributo.
