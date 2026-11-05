#!/usr/bin/env python3
"""Development only: generate oracle outputs with /usr/bin/xsltproc, offline."""
import pathlib, tempfile, re, subprocess, urllib.parse, xml.etree.ElementTree as ET
pkg = pathlib.Path(__file__).resolve().parents[1]
repo = pkg.parent
resources = pkg / 'resources'
locations = {
'4.0': 'www.sat.gob.mx/sitio_internet/cfd/4/cadenaoriginal_4_0/cadenaoriginal_4_0.xslt',
'3.3': 'www.sat.gob.mx/sitio_internet/cfd/3/cadenaoriginal_3_3/cadenaoriginal_3_3.xslt',
'2.0': 'www.sat.gob.mx/esquemas/retencionpago/2/retenciones.xslt',
'1.1': 'www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/cadenaoriginal_TFD_1_1.xslt'}
with tempfile.TemporaryDirectory() as tmp:
    tmp = pathlib.Path(tmp)
    for src in resources.rglob('*.xslt'):
        dest = tmp / src.relative_to(resources)
        dest.parent.mkdir(parents=True, exist_ok=True)
        def rewrite(m):
            u = urllib.parse.urlsplit(m[1].decode())
            return b'href="' + str(tmp / (u.netloc + u.path)).encode() + b'"'
        dest.write_bytes(re.sub(rb'href="(https?://[^"]+)"', rewrite, src.read_bytes()))
    files = sorted((repo/'testdata/cfdi').glob('*.xml')) + [repo/'testdata/retenciones/retenciones20.xml'] + sorted((pkg/'testdata/xml').glob('*.xml'))
    # Extract TFD as an independently namespaced document from every real invoice.
    for src in (repo/'testdata/cfdi').glob('*real.xml'):
        for tfd in ET.parse(src).iter('{http://www.sat.gob.mx/TimbreFiscalDigital}TimbreFiscalDigital'):
            dest = pkg/'testdata/xml'/('tfd-'+src.name)
            ET.ElementTree(tfd).write(dest, encoding='utf-8', xml_declaration=True)
            if dest not in files: files.append(dest)
    for src in files:
        root = ET.parse(src).getroot()
        version = root.get('Version', root.get('version'))
        out = subprocess.run(['/usr/bin/xsltproc', '--nonet', str(tmp/locations[version]), str(src)], check=True, capture_output=True).stdout
        (pkg/'testdata/golden'/(src.stem+'.txt')).write_bytes(out)
        print(src.name, len(out))
