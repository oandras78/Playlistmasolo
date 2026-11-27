package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCopyWeirdNames(t *testing.T) {
	tmp := t.TempDir()
	music := filepath.Join(tmp, "zene [2024] & #más")
	os.MkdirAll(filepath.Join(music, "almappa"), 0o755)
	names := []string{
		"01 - Árvíztűrő tükörfúrógép.mp3",
		"02 [live] & more #1.mp3",
		"03 日本語 😀.flac",
		"04 ő ű ŏ ã ñ.mp3",
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(music, n), []byte("adat "+n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(music, "almappa", "x.mp3"), []byte("x"), 0o644)
	old := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	os.Chtimes(filepath.Join(music, names[0]), old, old)

	var sb strings.Builder
	sb.WriteString("\ufeff#EXTM3U\r\n#EXTINF:1,x\r\n")
	for _, n := range names {
		sb.WriteString(filepath.Join(music, n) + "\r\n")
	}
	sb.WriteString("almappa/x.mp3\r\n") // relatív
	sb.WriteString(filepath.Join(music, "nincs.mp3") + "\r\n")
	sb.WriteString("http://example.com/stream\r\n")
	listPath := filepath.Join(music, "lista.m3u8")
	os.WriteFile(listPath, []byte(sb.String()), 0o644)

	dest := filepath.Join(tmp, "ki")
	var c counters
	var cancel atomic.Bool
	j := &job{Enc: encAuto, Log: func(s string) { t.Log(s) }, Progress: func(int, int) {}, Cancel: &cancel}
	j.run(listPath, dest, &c)

	if c.Copied != 5 || c.Missing != 1 || c.Skipped != 1 || c.Errors != 0 {
		t.Fatalf("counters: %+v", c)
	}
	for _, n := range append(names, "x.mp3") {
		b, err := os.ReadFile(filepath.Join(dest, n))
		if err != nil {
			t.Fatalf("hiányzik: %s: %v", n, err)
		}
		if n != "x.mp3" && string(b) != "adat "+n {
			t.Fatalf("rossz tartalom: %s", n)
		}
	}
	st, _ := os.Stat(filepath.Join(dest, names[0]))
	if !st.ModTime().Equal(old) {
		t.Fatalf("módosítási idő nem maradt meg: %v", st.ModTime())
	}

	// Második futás: minden "MEGVAN", semmi (2)
	c = counters{}
	j.run(listPath, dest, &c)
	if c.Copied != 0 || c.Existed != 5 {
		t.Fatalf("második futás: %+v", c)
	}
	files, _ := os.ReadDir(dest)
	if len(files) != 5 {
		t.Fatalf("fájlok száma: %d", len(files))
	}

	// Azonos nevű, de más tartalmú fájl: (2) kapja
	os.WriteFile(filepath.Join(dest, "x.mp3"), []byte("MÁS TARTALOM, MÁS MÉRET"), 0o644)
	c = counters{}
	j.run(listPath, dest, &c)
	if _, err := os.Stat(filepath.Join(dest, "x (2).mp3")); err != nil {
		t.Fatalf("x (2).mp3 hiányzik: %v", err)
	}
	// ismételt futás: nem keletkezik x (3).mp3
	c = counters{}
	j.run(listPath, dest, &c)
	if _, err := os.Stat(filepath.Join(dest, "x (3).mp3")); err == nil {
		t.Fatal("ismételt futásnál felesleges x (3).mp3 jött létre")
	}
}

func TestEncodingAndParsing(t *testing.T) {
	// UTF-16 LE BOM-mal
	u16 := []byte{0xFF, 0xFE, 'a', 0, '.', 0, 'm', 0, 'p', 0, '3', 0}
	e, label := loadEntries(u16, "/", encAuto)
	if len(e) != 1 || e[0] != "a.mp3" {
		t.Fatalf("utf16: %v %s", e, label)
	}
	// UTF-8 BOM nélkül
	e, _ = loadEntries([]byte("#EXTM3U\nárvíz.mp3\n\n  x.mp3  \n"), "/", encAuto)
	if len(e) != 2 || e[0] != "árvíz.mp3" || e[1] != "x.mp3" {
		t.Fatalf("utf8: %q", e)
	}
	// file:// URL
	p, ok := resolveEntry("file:///tmp/a%20b%5Bx%5D.mp3", "/lista")
	if !ok || p != "/tmp/a b[x].mp3" {
		t.Fatalf("file url: %q %v", p, ok)
	}
	if _, ok := resolveEntry("https://x.y/z.mp3", "/"); ok {
		t.Fatal("http nem helyi")
	}
}

func TestAnalyze(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a [1].mp3"), []byte("12345"), 0o644)
	os.WriteFile(filepath.Join(tmp, "ő.mp3"), []byte("123"), 0o644)
	os.MkdirAll(filepath.Join(tmp, "mappa"), 0o755)
	list := "#EXTM3U\n#EXTINF:1,x\na [1].mp3\nő.mp3\nnincs.mp3\nmappa\nhttp://x/y.mp3\n"
	lp := filepath.Join(tmp, "l.m3u8")
	os.WriteFile(lp, []byte(list), 0o644)

	a := analyzeList(lp, encAuto)
	if a.Total() != 5 || a.Found != 2 || a.NotFound != 1 || a.Dirs != 1 || a.NotLocal != 1 || a.Errs != 0 || a.Bytes != 8 {
		t.Fatalf("%+v", a)
	}
	want := []int{stOK, stOK, stMissing, stDir, stNotLocal}
	for i, w := range want {
		if a.Entries[i].Status != w {
			t.Fatalf("%d. bejegyzés: %d, várt: %d", i, a.Entries[i].Status, w)
		}
	}
	t.Log(a.Summary())
	if b := analyzeList(filepath.Join(tmp, "nincs.m3u"), encAuto); b.Err == "" {
		t.Fatal("hiányzó lista hibát kellene adjon")
	}
}

// A kritikus hiba: azonos név + azonos méret + azonos módosítási idő, de MÁS tartalom.
func TestSameSizeSameTimeDifferentContentIsNotSkipped(t *testing.T) {
	tmp := t.TempDir()
	srcDir, dest := filepath.Join(tmp, "src"), filepath.Join(tmp, "ki")
	os.MkdirAll(srcDir, 0o755)
	os.MkdirAll(dest, 0o755)
	when := time.Date(2022, 5, 6, 7, 8, 9, 0, time.UTC)

	src := filepath.Join(srcDir, "Track.mp3")
	os.WriteFile(src, []byte("AAAAAAAAAAAAAAAA"), 0o644)
	os.Chtimes(src, when, when)
	old := filepath.Join(dest, "Track.mp3")
	os.WriteFile(old, []byte("BBBBBBBBBBBBBBBB"), 0o644) // ugyanakkora, más tartalom
	os.Chtimes(old, when, when)

	lp := filepath.Join(srcDir, "l.m3u8")
	os.WriteFile(lp, []byte(src+"\n"), 0o644)
	var c counters
	j := &job{Log: func(s string) { t.Log(s) }, Progress: func(int, int) {}}
	j.run(lp, dest, &c)

	if c.Copied != 1 || c.Existed != 0 {
		t.Fatalf("a más tartalmú fájlt nem másolta át: %+v", c)
	}
	if b, _ := os.ReadFile(old); string(b) != "BBBBBBBBBBBBBBBB" {
		t.Fatal("a meglévő fájlt felülírta!")
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "Track (2).mp3")); string(b) != "AAAAAAAAAAAAAAAA" {
		t.Fatal("Track (2).mp3 hiányzik vagy hibás")
	}
}

func TestSameContentAndVerify(t *testing.T) {
	tmp := t.TempDir()
	big := make([]byte, 3<<20+17) // több puffernyi + maradék
	for i := range big {
		big[i] = byte(i * 7)
	}
	a, b, c := filepath.Join(tmp, "a"), filepath.Join(tmp, "b"), filepath.Join(tmp, "c")
	os.WriteFile(a, big, 0o644)
	os.WriteFile(b, big, 0o644)
	big2 := append([]byte(nil), big...)
	big2[len(big2)-1] ^= 1 // csak az utolsó bájt tér el
	os.WriteFile(c, big2, 0o644)

	if same, err := sameContent(a, b); err != nil || !same {
		t.Fatalf("azonos tartalmat kellene lásson: %v %v", same, err)
	}
	if same, err := sameContent(a, c); err != nil || same {
		t.Fatalf("az utolsó bájtban eltérőt nem vette észre: %v %v", same, err)
	}
	st, _ := os.Stat(a)
	dst := filepath.Join(tmp, "d")
	if err := copyFile(a, dst, st, true); err != nil {
		t.Fatal(err)
	}
	if same, _ := sameContent(a, dst); !same {
		t.Fatal("a másolat eltér")
	}
}

func TestResolveTargetErrorDoesNotLoopForever(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "x.mp3")
	os.WriteFile(src, []byte("x"), 0o644)
	// a "cél könyvtár" valójában egy fájl: az Lstat hibát ad (ENOTDIR), nem ErrNotExist
	notDir := filepath.Join(tmp, "fajl")
	os.WriteFile(notDir, []byte("x"), 0o644)
	done := make(chan error, 1)
	go func() { _, _, err := resolveTarget(notDir, "x.mp3", src); done <- err }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("végtelen ciklus")
	}
}

func TestAutoEncodingPrefersListThatMatchesFiles(t *testing.T) {
	tmp := t.TempDir()
	// A fájl neve "ő" (U+0151). Windows-1250-ben 0xF5, ami önmagában nem érvényes UTF-8.
	// Ez a teszt Linuxon a "ANSI" dekódolót egyszerű latin-1-ként kezeli (other.go), ezért itt csak
	// azt ellenőrizzük, hogy érvényes UTF-8 listánál a megtalálható fájl dönt.
	os.WriteFile(filepath.Join(tmp, "ő.mp3"), []byte("x"), 0o644)
	lst := []byte(filepath.Join(tmp, "ő.mp3") + "\n")
	e, label := loadEntries(lst, tmp, encAuto)
	if len(e) != 1 || !strings.Contains(label, "UTF-8") {
		t.Fatalf("%v %s", e, label)
	}
}
