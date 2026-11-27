@echo off
rem Windows exe forditasa. Kell: Go 1.21+ (https://go.dev/dl/), opcionalisan Python 3 + Pillow (pip install pillow)
setlocal
cd /d "%~dp0"
set VER=%1
if "%VER%"=="" set VER=1.2.0
python tools\make_res.py %VER% rsrc_windows_amd64.syso || echo Python/Pillow nincs: a meglevo rsrc_windows_amd64.syso marad
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-H windowsgui -s -w -X main.version=%VER%" -o PlaylistMasolo.exe .
if errorlevel 1 (echo HIBA a forditasnal & exit /b 1)
echo Kesz: PlaylistMasolo.exe (%VER%)
