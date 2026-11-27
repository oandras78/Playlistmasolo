# Lejátszási lista másoló

M3U/M3U8 listákban szereplő fájlok másolása egy megadott mappába, az eredeti fájlnevekkel.
Bármilyen karakterrel működik a fájlnevekben (Unicode-os Windows API-k, nincs kódolási gond).

## Funkciók
- Lista behúzása / hozzáadása után azonnal megmutatja a tartalmát: hány bejegyzés van benne, ebből hányat tudott beolvasni (megtalált fájl), hány hiányzik, és a méretüket.
- Soronként látszik az állapot (OK / HIÁNYZIK / NEM HELYI / MAPPA / HIBA); szűrhető a hiányzókra.
- Több lista egyszerre, összesítéssel; kódolás automatikus felismerése vagy kézi választása (a tartalom ilyenkor újraszámolódik).
- Másolás eredeti fájlnevekkel, almappa listánként (opcionális), napló a célmappában.
- Azonos nevű fájl a célmappában: a program a TARTALMAT hasonlítja össze (bájtonként), nem a méretet/dátumot. Azonos tartalomnál kihagyja ("MEGVAN"), eltérőnél `név (2)` néven menti – felülírás soha nincs, és ismételt futtatásnál nem halmozódnak a másolatok.
- Opcionális "Ellenőrzés másolás után": a másolat tartalmát visszaolvasva összeveti a forrással; eltérésnél törli a másolatot és hibát jelez. (Méretellenőrzés mindig van.)
- A befejezés állapota külön szerepel a naplóban: KÉSZ / BEFEJEZVE (nem minden bejegyzés másolható / hibákkal) / MEGSZAKÍTVA / LEÁLLT. A naplófájl mentési hibáját is jelzi.

## Fájlok
- `core.go` – lista beolvasása, kódolás-felismerés, útvonal-feloldás, másolás (platformfüggetlen, tesztelhető)
- `analyze.go` – a lista elemzése másolás nélkül (bejegyzések állapota, összegzés)
- `core_test.go` – egységtesztek (`go test ./...`)
- `decode_windows.go` / `other.go` – ANSI kódlapok dekódolása (Windows: MultiByteToWideChar; máshol csak a tesztekhez)
- `gui_windows.go` – az ablakos felület (tiszta Win32, külső függőség nélkül); itt a `version` változó a verziószám
- `rsrc_windows_amd64.syso` – beágyazott ikon, verzióinfó, manifest (a `tools/make_res.py` állítja elő)
- `tools/make_res.py` – az ikont (Pillow-val rajzolja), a verzióinfót és a manifestet generálja .syso-ba
- `build.sh` / `build.bat` – fordítás

## Fordítás
Csak Go 1.21+ kell: `build.bat` (Windows) vagy `./build.sh` (Linux/macOS), opcionális argumentum a verzió, pl. `build.bat 1.2.0`.
Verzió- vagy ikonváltáshoz Python 3 + Pillow kell (`pip install pillow`); nélküle a mellékelt .syso marad.
Kézzel: `go build -trimpath -ldflags "-H windowsgui -s -w -X main.version=1.2.0" -o PlaylistMasolo.exe .` (GOOS=windows)

Futtatáshoz Windows 10 vagy újabb kell.
