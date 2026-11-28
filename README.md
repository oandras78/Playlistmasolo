# Playlist Másoló | Playlist Copier

[🇭🇺 Magyar](#magyar) | [🇬🇧 English](#english)

A Windows segédprogram M3U/M3U8 lejátszási listákban szereplő helyi zenefájlok másolására, az eredeti fájlnevek megtartásával.

A Windows utility for copying local audio files referenced by M3U/M3U8 playlists while preserving their original filenames.

---

<a id="magyar"></a>\n## Magyar

### Funkciók

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

### Letöltés

[Legújabb kiadás letöltése](https://github.com/oandras78/Playlistmasolo/releases/latest)

---

<a id="english"></a>\n## English

### Features

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

### Download

[Download the latest release](https://github.com/oandras78/Playlistmasolo/releases/latest)
