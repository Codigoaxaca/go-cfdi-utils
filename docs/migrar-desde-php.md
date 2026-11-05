# Equivalencias con la versión PHP (eclipxe/CfdiUtils)

Si ya conoces `CfdiUtils` de PHP, esta tabla te ayuda a encontrar lo equivalente en Go.

## Convenciones

- Los nombres de dominio del SAT se conservan en español (`Comprobante`, `Emisor`, `Sello`,
  `CadenaDeOrigen`), los métodos siguen el estilo de Go (`PascalCase`, sin prefijo `get`).
- Los arreglos de atributos de PHP son `cfdi40.Attrs` (`map[string]string`).
- `$node['Atributo']` es `node.Attr("Atributo")`; `$node['Atributo'] = 'x'` es `node.SetAttr("Atributo", "x")`.
- Las excepciones son valores `error`.
- Las operaciones que pueden usar la red reciben (o tienen una variante con) `context.Context`.

## Clases y paquetes

| PHP | Go |
|-----|----|
| `CfdiUtils\CfdiCreator40` | `cfdi40.Creator` (`cfdi40.NewCreator`) |
| `CfdiUtils\CfdiValidator40` | `cfdi40.Validator` (`cfdi40.NewValidator`) |
| `CfdiUtils\Elements\Cfdi40\*` | `cfdi40.Comprobante`, `cfdi40.Emisor`, `cfdi40.Concepto`, ... |
| `CfdiUtils\Elements\Pagos20\*` | `complementos/pagos20` |
| `CfdiUtils\SumasPagos20\Calculator` | `pagos20` (calculadora, ver [pagos 2.0](crear/pagos20.md)) |
| `CfdiUtils\Elements\ImpLocal10\*` | `complementos/implocal10` |
| `CfdiUtils\Elements\LeyendasFiscales10\*` | `complementos/leyendasfisc10` |
| `CfdiUtils\Elements\CartaPorte31\*` | `complementos/cartaporte31` |
| `CfdiUtils\Elements\Tfd11\TimbreFiscalDigital` | `complementos/tfd11` |
| `CfdiUtils\Nodes\Node`, `XmlNodeUtils` | `nodes.Node`, `nodes.ParseXML`, `node.XML()` |
| `CfdiUtils\Certificado\Certificado` | `credentials.Certificate` |
| `CfdiUtils\PemPrivateKey\PemPrivateKey` | `credentials.PrivateKey` (lee el `.key` DER del SAT directamente) |
| `phpcfdi/credentials` `Credential` | `credentials.Credential` |
| `CfdiUtils\Certificado\SerialNumber` | `credentials.SerialNumber` |
| `CfdiUtils\Certificado\NodeCertificado` | `credentials.ParseCertificate([]byte(comprobante.Attr("Certificado")))` |
| `CfdiUtils\CadenaOrigen\DOMBuilder` | `cadenaorigen.XsltBuilder` (Go puro, sin `ext-xsl`) |
| `CfdiUtils\TimbreFiscalDigital\TfdCadenaDeOrigen` | `cadenaorigen.BuildTFD` |
| `CfdiUtils\XmlResolver\XmlResolver` | `xmlresolver.Resolver` |
| `CfdiUtils\ConsultaCfdiSat\WebService` | `estadosat.WebService` |
| `CfdiUtils\ConsultaCfdiSat\RequestParameters` | `estadosat.RequestParameters` |
| `CfdiUtils\ConsultaCfdiSat\StatusResponse` | `estadosat.StatusResponse` |
| `CfdiUtils\SumasConceptos\SumasConceptos` | `sumasconceptos.SumasConceptos` |
| `CfdiUtils\SumasConceptos\SumasConceptosWriter` | `cfdi40.WriteSumasConceptos` |
| `CfdiUtils\Validate\Asserts`, `Assert`, `Status` | `validate.Asserts`, `validate.Assert`, `validate.Status` |
| `CfdiUtils\Validate\MultiValidator` | `validate.MultiValidator` |
| `CfdiUtils\Validate\Xml\XmlFollowSchema` | `cfdi40validators.XmlFollowSchema` + `xmlschema.XmllintValidator` |
| `CfdiUtils\Cleaner\Cleaner` | `cleaner.Cleaner`, `cleaner.Clean` |
| `CfdiUtils\Utils\RegimenCapitalRemover` | `credentials.RemoveRegimenCapital` |

## Métodos frecuentes

| PHP | Go |
|-----|----|
| `new CfdiCreator40($attrs, $certificado)` | `creator := cfdi40.NewCreator(attrs)`; `creator.PutCertificado(cert, true)` |
| `$creator->comprobante()` | `creator.Comprobante()` |
| `$comprobante->getEmisor()` | `comprobante.Emisor()` |
| `$comprobante->addEmisor([...])` | `comprobante.AddEmisor(cfdi40.Attrs{...})` |
| `$comprobante->addConcepto([...])->addTraslado([...])` | `comprobante.AddConcepto(cfdi40.Attrs{...}).AddTraslado(cfdi40.Attrs{...})` |
| `$concepto->multiInformacionAduanera([...], [...])` | `concepto.MultiInformacionAduanera(cfdi40.Attrs{...}, cfdi40.Attrs{...})` |
| `$comprobante->addComplemento($pagos)` | `comprobante.AddComplemento(pagos)` |
| `$creator->addSumasConceptos(null, 2)` | `creator.AddSumasConceptos(nil, 2)` |
| `$creator->addSello('file://key.pem', 'pass')` | `creator.AddSello(key)` |
| `$creator->moveSatDefinitionsToComprobante()` | `creator.MoveSatDefinitionsToComprobante()` |
| `$creator->validate()` | `creator.Validate()` |
| `$creator->asXml()` / `saveXml($file)` | `creator.AsXML()` / `creator.SaveXML(file)` |
| `$creator->buildCadenaDeOrigen()` | `creator.BuildCadenaDeOrigen()` |
| `$asserts->hasErrors()` / `errors()` | `asserts.HasErrors()` / `asserts.Errors()` |
| `$assert->getCode()`, `getTitle()`, `getStatus()`, `getExplanation()` | `a.Code`, `a.Title`, `a.Status`, `a.Explanation` |
| `$certificado->getRfc()`, `getName()`, `getSerial()` | `cert.Rfc()`, `cert.Name()`, `cert.NoCertificado()` |
| `$certificado->getValidFrom()` (timestamp) | `cert.ValidFrom()` (`time.Time`) |
| `$certificado->belongsTo($pemKeyFile, $pass)` | `key.BelongsTo(cert)` |
| `$certificado->verify($data, $sig)` | `cert.Verify(data, sig)` |
| `$node->searchNode('cfdi:Complemento', 'tfd:TimbreFiscalDigital')` | `node.SearchNode("cfdi:Complemento", "tfd:TimbreFiscalDigital")` |
| `$node->searchAttribute('cfdi:Emisor', 'Rfc')` | `node.SearchAttribute("cfdi:Emisor", "Rfc")` |
| `RequestParameters::createFromCfdi($cfdi)` | `estadosat.RequestParametersFromXML(xml)` |
| `$response->getCfdi()` | `resp.Estado` |
| `$response->getCancellable()` | `resp.EsCancelable` |
| `$response->getCancellationStatus()` | `resp.EstatusCancelacion` |
| `$response->getValidationEfos()` | `resp.ValidacionEFOS` |
| `Cleaner::staticClean($xml)` | `cleaner.Clean(xml)` |

## Lo que no se portó (por ahora)

- Creación de CFDI 3.3 (`CfdiCreator33`) y sus validaciones: se puede **leer**, consultar su
  estado y generar su cadena original, pero no hay creador ni validadores para 3.3.
- CFDI de Retenciones (`RetencionesCreator20`): solo se genera la cadena original.
- Elementos de complementos menos frecuentes (Nómina 1.2, Comercio Exterior 2.0, INE, IEDU,
  donatarias, etc.): se pueden construir con [`nodes`](componentes/nodos.md); la cadena original
  ya los soporta porque los XSLT del SAT vienen incluidos.
- `QuickReader`: usa `SearchNode`/`SearchAttribute` o los elementos de `cfdi40`.
- Los generadores de cadena alternativos (`SaxonbCliBuilder`, `GenkgoXslBuilder`): no son
  necesarios; implementa `cadenaorigen.Builder` si quieres usar otro.
