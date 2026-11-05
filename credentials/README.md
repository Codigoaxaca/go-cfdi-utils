# Credenciales SAT en Go puro

```go
cred, err := credentials.OpenFiles("certificado.cer", "privada.key", "contraseña")
if err != nil {
    return err
}
sello, err := cred.SignBase64([]byte(cadenaOriginal))
```

Sólo usa la biblioteca estándar (Go 1.24 o superior). Abre certificados DER,
PEM con bloques adicionales y base64, incluso el atributo `Comprobante@Certificado`
con espacios y saltos de línea. No ejecuta OpenSSL.

Las llaves RSA pueden ser PKCS#1 o PKCS#8 sin cifrar, en DER o PEM; PKCS#8
cifrado acepta DER, PEM `ENCRYPTED PRIVATE KEY` o base64. Se admiten:

- PBES2/PBKDF2 con HMAC-SHA1 (incluida la PRF predeterminada) o HMAC-SHA256,
  y CBC con DES, 3DES o AES de 128, 192 y 256 bits.
- PKCS#12 PBE `pbeWithSHAAnd3-KeyTripleDES-CBC`, con contraseña UTF-16BE.
- PEM tradicional con `Proc-Type`/`DEK-Info`, mediante `x509.DecryptPEMBlock`.

`errors.Is(err, credentials.ErrInvalidPassword)` identifica errores de contraseña
por relleno o contenido ASN.1 descifrado inválidos. El cifrado CBC sin autenticación
no permite distinguirlos de un archivo cifrado dañado. Una KDF, PRF o cifra no
soportada produce un error descriptivo distinto. Se limitan las iteraciones de
KDF a 10 millones para evitar trabajo desmedido con entradas malformadas.
`PrivateKey.PEM()` devuelve material secreto sin cifrar.

`Sign` y `Verify` usan RSA PKCS#1 v1.5 con SHA-256. Las variantes `WithHash`
aceptan hashes disponibles; SHA1, SHA256, SHA384 y SHA512 están enlazados.

## Lectura y clasificación de certificados

`X509()` devuelve el certificado estándar. Si éste rechaza una estructura antigua,
se intenta leer con ASN.1 los campos necesarios y la llave pública; en ese caso
`X509()` es `nil`. No se validan la firma del certificado, cadena de confianza,
autorización SAT ni revocación. `IsValidAt` comprueba únicamente fechas, con ambos
extremos incluidos.

La clasificación solicitada se basa en `keyUsage`: `IsCsd` requiere
`digitalSignature` y `nonRepudiation` (`ContentCommitment` en Go), sin
`dataEncipherment` ni `keyAgreement`; `IsFiel` detecta estos últimos usos.
Un certificado sin esos perfiles puede devolver falso en ambos métodos.
Es una heurística de perfil SAT, no una validación de autorización.
La [implementación PHP consultada](https://github.com/phpcfdi/credentials/blob/main/src/Certificate.php)
usa en cambio la presencia de unidad organizacional (`OU`) para clasificar.

`Name` prioriza el atributo `name` (2.5.4.41), luego organización y CN.
`LegalNameForCfdi40` quita el régimen de capital sólo cuando el RFC tiene doce
caracteres. Se porta la lista completa de sufijos de
`CfdiUtils/Utils/RegimenCapitalRemover.php`, con comparación sin distinguir
mayúsculas; los paréntesis y demás caracteres se interpretan literalmente.
