# Documentación de go-cfdi-utils

`go-cfdi-utils` es una librería en Go para crear, sellar, leer y validar CFDI 4.0 de México, y
para consultar su estado en el SAT. Está inspirada en la librería PHP
[eclipxe/CfdiUtils](https://github.com/eclipxe13/CfdiUtils).

## Inicio

- [Instalación](instalacion.md)
- [Equivalencias con la versión PHP](migrar-desde-php.md)

## Crear CFDI

- [Crear un CFDI 4.0](crear/crear-cfdi-40.md) — el punto de partida
- [Elementos de CFDI 4.0](crear/elementos-cfdi-40.md) — referencia de `Comprobante`, `Concepto`, etc.
- [Complementos](crear/complementos.md) — Pagos 2.0, impuestos locales, leyendas fiscales, Carta Porte 3.1
- [Complemento de pagos 2.0 y su calculadora](crear/pagos20.md)

## Leer CFDI

- [Leer un CFDI](leer/leer-cfdi.md)
- [Limpieza de un CFDI](leer/limpieza-cfdi.md)

## Validar CFDI

- [Validación de CFDI 4.0](validar/validacion-cfdi-40.md) — creados y recibidos, `Asserts`, validadores propios
- [Validaciones de CFDI 4.0](validar/validaciones-40.md) — lista de códigos
- [Validación contra esquemas XSD](validar/esquemas-xsd.md)

## Componentes

- [Certificados y llaves privadas](componentes/certificados.md)
- [Consulta del estado de un CFDI en el SAT](componentes/estado-sat.md)
- [Cadena de origen](componentes/cadena-de-origen.md)
- [Estructura de datos `nodes`](componentes/nodos.md)
- [Recursos del SAT (XSLT y XSD)](componentes/recursos-sat.md) — descarga, actualización mensual, forzar y borrar caché, uso sin internet
- [Certificados del SAT (`xmlresolver`)](componentes/xmlresolver.md)

## Utilerías

- [Sumas de conceptos](utilerias/sumas-conceptos.md)
