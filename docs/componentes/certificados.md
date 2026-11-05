# Certificados y llaves privadas (`credentials`)

El paquete `credentials` trabaja con los archivos del Certificado de Sello Digital (CSD) y de la
e.firma (FIEL) **tal como los entrega el SAT**: el certificado `.cer` y la llave privada `.key`
cifrada con contraseña. No hace falta convertirlos a PEM ni tener OpenSSL instalado.

```go
import "github.com/codigoaxaca/go-cfdi-utils/credentials"
```

## Credencial: certificado + llave privada

```go
csd, err := credentials.OpenFiles("EKU9003173C9.cer", "EKU9003173C9.key", "12345678a")
if err != nil {
	// archivo inexistente, contraseña incorrecta o la llave no corresponde al certificado
	log.Fatal(err)
}

fmt.Println(csd.Rfc())           // EKU9003173C9
fmt.Println(csd.Name())          // ESCUELA KEMPER URGATE SA DE CV
fmt.Println(csd.NoCertificado()) // 30001000000500003416
fmt.Println(csd.IsCsd())         // true

// firmar (RSA PKCS#1 v1.5 + SHA-256, el algoritmo del sello de CFDI)
sello, err := csd.SignBase64([]byte(cadenaOriginal))
firma, err := csd.Sign(datos)
ok := csd.Verify(datos, firma)
```

Otras formas de crearla:

```go
csd, err := credentials.ParseCredential(cerBytes, keyBytes, "contraseña") // desde bytes (base de datos, S3, ...)
csd, err := credentials.NewCredential(certificado, llave)                // desde objetos ya abiertos
```

`OpenFiles`, `ParseCredential` y `NewCredential` devuelven error si la llave privada no corresponde
al certificado.

## Certificado

```go
cert, err := credentials.OpenCertificateFile("EKU9003173C9.cer")
cert, err := credentials.ParseCertificate(data)
```

`ParseCertificate` acepta:

- El archivo `.cer` (DER).
- Un PEM (`-----BEGIN CERTIFICATE-----`), aunque contenga otros bloques.
- El base64 en una sola línea, como el atributo `Comprobante@Certificado` de un CFDI
  (se toleran espacios y saltos de línea).

### Información del certificado

| Método | Ejemplo | Descripción |
|--------|---------|-------------|
| `Rfc()` | `EKU9003173C9` | RFC del titular (atributo `x500UniqueIdentifier`). |
| `RfcRepresentante()` | | RFC del representante legal (personas morales), si existe. |
| `Curp()` | | CURP (atributo `serialNumber`), si existe. |
| `Name()` | `ESCUELA KEMPER URGATE SA DE CV` | Nombre o razón social. |
| `NameWithoutRegimenCapital()` | `ESCUELA KEMPER URGATE` | Nombre sin el régimen de capital. |
| `LegalNameForCfdi40()` | `ESCUELA KEMPER URGATE` | El nombre que exige CFDI 4.0 en `Emisor@Nombre`: sin régimen de capital para personas morales (RFC de 12 caracteres), el nombre completo para personas físicas. |
| `IsPersonaMoral()` | `true` | RFC de 12 caracteres. |
| `NoCertificado()` | `30001000000500003416` | Número de certificado (el valor de `Comprobante@NoCertificado`). |
| `SerialNumber()` | | Número de serie en sus distintas representaciones (ver abajo). |
| `ValidFrom()`, `ValidTo()` | | Vigencia (`time.Time`). |
| `IsValidAt(t)` | | Si la fecha está dentro de la vigencia. |
| `IsCsd()`, `IsFiel()` | | Tipo de certificado, según los usos de llave (`keyUsage`). |
| `Issuer()` | | Emisor del certificado (la autoridad certificadora del SAT). |
| `DER()`, `PEM()`, `Base64()` | | El certificado en distintos formatos. `Base64()` es el valor de `Comprobante@Certificado`. |
| `PublicKeyPEM()` | | La llave pública en PEM. |
| `X509()` | | El `*x509.Certificate` de la biblioteca estándar. |
| `SaveFile(path)` | | Guarda el certificado en formato DER (`.cer`). |

### Verificar una firma

```go
ok := cert.Verify([]byte(cadenaOriginal), firma)                       // RSA-SHA256
ok := cert.VerifyWithHash(crypto.SHA1, datos, firma)                   // otro algoritmo
```

### Números de serie

El número de certificado que se usa en los CFDI es la representación ASCII del número de serie.
`SerialNumber()` lo entrega en tres formatos:

```go
s := cert.SerialNumber()
s.Ascii()       // 30001000000500003416  (igual a NoCertificado())
s.Hexadecimal() // 3330303031303030303030353030303033343136
s.Decimal()     // 292233162870206001759766198462772978647764840758
```

También se pueden crear con `credentials.SerialNumberFromHex`, `SerialNumberFromDecimal` y
`SerialNumberFromAscii`.

## Llave privada

```go
key, err := credentials.OpenPrivateKeyFile("EKU9003173C9.key", "12345678a")
key, err := credentials.ParsePrivateKey(data, "12345678a")
if errors.Is(err, credentials.ErrInvalidPassword) {
	// contraseña incorrecta
}

firma, err := key.Sign(datos)    // RSA PKCS#1 v1.5 + SHA-256
key.BelongsTo(cert)              // si corresponde al certificado
pem := key.PEM()                 // PKCS#8 PEM SIN cifrar, ¡cuidado!
rsaKey := key.RSA()              // *rsa.PrivateKey
```

Formatos soportados:

- El archivo `.key` del SAT: PKCS#8 cifrado en DER (PBES2 + PBKDF2 + 3DES).
- PKCS#8 cifrado en PEM (`ENCRYPTED PRIVATE KEY`), con PBES2 (HMAC-SHA1/SHA256; DES, 3DES,
  AES-128/192/256) o PKCS#12 PBE (`pbeWithSHAAnd3-KeyTripleDES-CBC`).
- PKCS#8 o PKCS#1 sin cifrar, en DER o PEM (`PRIVATE KEY`, `RSA PRIVATE KEY`).
- PEM tradicional cifrado (`Proc-Type: 4,ENCRYPTED`).
- Cualquiera de los anteriores en base64 de una sola línea.

## Certificados del SAT (Timbre Fiscal Digital)

El atributo `NoCertificadoSAT` del timbre indica el certificado con el que el SAT (o el PAC)
selló el timbre. Estos certificados se publican en `https://rdc.sat.gob.mx/`:

```go
credentials.IsValidNoCertificado("00001000000505211329") // true (20 dígitos)
url, err := credentials.SatCertificateURL("00001000000505211329")
// https://rdc.sat.gob.mx/rccf/000010/000005/05/21/13/00001000000505211329.cer
```

Para descargarlo (con caché) usa `xmlresolver.Resolver.SatCertificate`
(ver [xmlresolver](xmlresolver.md)).

## Extraer el certificado de un CFDI

```go
c, err := cfdi40.ReadFile("cfdi.xml")
cert, err := credentials.ParseCertificate([]byte(c.Attr("Certificado")))
fmt.Println(cert.Rfc(), cert.NoCertificado(), cert.ValidTo())
err = cert.SaveFile(cert.NoCertificado() + ".cer")
```

## Régimen de capital

```go
credentials.RemoveRegimenCapital("ESCUELA KEMPER URGATE SA DE CV") // ESCUELA KEMPER URGATE
```

Usa la misma lista de sufijos que la versión PHP (`SA DE CV`, `S DE RL DE CV`, `SAPI DE CV`,
`SC`, `AC`, ...), sin distinguir mayúsculas.

## Comandos útiles de OpenSSL

No son necesarios para usar la librería, pero son útiles para inspeccionar archivos:

```shell
# información del certificado
openssl x509 -nameopt utf8,sep_multiline,lname -inform DER -noout -dates -serial \
  -subject -fingerprint -pubkey -in EKU9003173C9.cer

# convertir la llave privada a PEM sin contraseña
openssl pkcs8 -inform DER -in EKU9003173C9.key -out EKU9003173C9.key.pem
```
