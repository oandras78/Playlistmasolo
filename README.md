# Playlist Másoló | Playlist Copier

**Magyar | English**

A Windows segédprogram M3U/M3U8 lejátszási listákban szereplő helyi zenefájlok másolására, az eredeti fájlnevek megtartásával.

The Windows utility copies local audio files referenced by M3U/M3U8 playlists while preserving their original filenames.

## Funkciók | Features

### Magyar

- M3U/M3U8 listák hozzáadása vagy behúzása, a tartalom azonnali elemzésével.
- Összesítés a bejegyzések számáról, a megtalált és hiányzó fájlokról, valamint a fájlméretekről.
- Bejegyzésenkénti állapot: **OK / HIÁNYZIK / NEM HELYI / MAPPA / HIBA**.
- A hiányzó fájlok szűrése.
- Több lejátszási lista egyidejű kezelése és összesített eredmény.
- Automatikus kódolásfelismerés vagy kézi kódolásválasztás.
- Másolás az eredeti fájlnevekkel, opcionálisan listánként külön almappába.
- Napló mentése a célmappába.
- Névütközés esetén a fájlok tartalmát bájtonként hasonlítja össze, nem csak a méretet vagy dátumot.
- Azonos tartalom esetén kihagyja a fájlt; eltérő tartalom esetén új nevet ad, például `track (2).mp3`. Meglévő fájlokat soha nem ír felül.
- Opcionális másolás utáni tartalomellenőrzés. Eltérés esetén törli a másolatot és hibát jelez. Méretellenőrzés mindig van.
- A befejezési állapot és a naplófájl mentésének esetleges hibája is megjelenik.

### English

- Add or drag-and-drop M3U/M3U8 playlists and analyze their contents immediately.
- Summary of entries, found and missing files, and file sizes.
- Per-entry status: **OK / MISSING / NOT LOCAL / FOLDER / ERROR**.
- Filter missing files.
- Process multiple playlists at once and view a combined summary.
- Automatic encoding detection or manual encoding selection.
- Copy files using their original filenames, optionally into a separate subfolder for each playlist.
- Save a log in the destination folder.
- On filename conflicts, compare file contents byte by byte—not just file size or date.
- Skip files with identical content. If content differs, save under a new name, such as `track (2).mp3`. Existing files are never overwritten.
- Optional content verification after copying. If a mismatch is detected, the copy is deleted and an error is reported. File size is always checked.
- The final operation status and any log-saving error are reported.

## Letöltés | Download

[Legújabb kiadás letöltése | Download the latest release](https://github.com/oandras78/Playlistmasolo/releases/latest)

## Rendszerkövetelmények | Requirements

- Windows 10 vagy újabb.
- A kiadott program futtatásához Go telepítése nem szükséges.

- Windows 10 or later.
- Go is not required to run the released executable.

## Fordítás forráskódból | Build from source

A fordításhoz Go 1.21 vagy újabb szükséges.

Go 1.21 or later is required to build the project.

Windows:
```bat
build.bat
```

Linux/macOS:
```bash
./build.sh
```

Opcionális verziószám is megadható, például `build.bat 1.2.0`.

An optional version number can be provided, for example `build.bat 1.2.0`.

A verzió- vagy ikonforrás frissítéséhez Python 3 és Pillow szükséges:
```bash
pip install pillow
```

Python 3 and Pillow are required only to regenerate version or icon resources:
```bash
pip install pillow
```

## Tesztek | Tests

Go egységtesztek:
```bash
go test ./...
```

Run the Go unit tests:
```bash
go test ./...
```

## Hibajelentés | Issues

Hibák és javaslatok jelentéséhez használd a GitHub Issues oldalt.

Please use GitHub Issues for bug reports and suggestions.

---

**Készítette: DJMTools | Created by DJMTools**
