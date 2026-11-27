#!/bin/sh
# Windows exe fordítása (Linuxon/macOS-en is). Kell: Go 1.21+, opcionálisan Python 3 + Pillow az erőforrásokhoz.
set -e
cd "$(dirname "$0")"
VER="${1:-1.2.0}"
if python3 -c "import PIL" 2>/dev/null; then
  python3 tools/make_res.py "$VER" rsrc_windows_amd64.syso
else
  echo "Pillow nincs: a meglévő rsrc_windows_amd64.syso marad (a beágyazott verzióinfó nem frissül)"
fi
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-H windowsgui -s -w -X main.version=$VER" -o PlaylistMasolo.exe .
echo "Kész: PlaylistMasolo.exe ($VER)"
