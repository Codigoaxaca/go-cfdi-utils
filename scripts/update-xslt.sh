#!/usr/bin/env bash
# Refresh the SAT stylesheets and recursive includes. Requires Python 3 (development only).
set -euo pipefail
cd "$(dirname "$0")/.."
python3 - <<'PY'
import pathlib, urllib.request, urllib.parse, xml.etree.ElementTree as ET
root = pathlib.Path('cadenaorigen/resources')
urls = [
'http://www.sat.gob.mx/sitio_internet/cfd/4/cadenaoriginal_4_0/cadenaoriginal_4_0.xslt',
'http://www.sat.gob.mx/sitio_internet/cfd/3/cadenaoriginal_3_3/cadenaoriginal_3_3.xslt',
'http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/cadenaoriginal_TFD_1_1.xslt',
'http://www.sat.gob.mx/esquemas/retencionpago/2/retenciones.xslt']
seen = set()
def fetch(url):
    u = urllib.parse.urlsplit(url)
    key = u.netloc + u.path
    if key in seen: return
    seen.add(key)
    path = root / key
    print(url, flush=True)
    request = urllib.request.Request(urllib.parse.urlunsplit(('https',u.netloc,u.path,u.query,'')), headers={'User-Agent':'Mozilla/5.0'})
    data = urllib.request.urlopen(request, timeout=90).read()
    doc = ET.fromstring(data)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(data)
    for inc in doc.findall('{http://www.w3.org/1999/XSL/Transform}include'):
        fetch(urllib.parse.urljoin(url, inc.attrib['href']))
for url in urls: fetch(url)
PY
