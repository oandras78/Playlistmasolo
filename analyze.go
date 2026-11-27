package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Egy bejegyzés állapota a lista elemzésekor.
const (
	stOK       = iota // a fájl megvan
	stMissing         // nem létezik
	stNotLocal        // nem helyi fájl (pl. http stream)
	stDir             // mappa, nem fájl
	stError           // más hiba (pl. nincs jogosultság)
)

type entryInfo struct {
	Raw    string // a lista sora, ahogy beolvastuk
	Path   string // feloldott teljes útvonal
	Status int
	Size   int64
}

type analysis struct {
	Path     string
	Enc      string
	Entries  []entryInfo
	Found    int   // megtalált fájlok
	NotFound int   // valóban hiányzó fájlok
	NotLocal int   // nem helyi bejegyzések (pl. internetes stream)
	Dirs     int   // mappák
	Errs     int   // egyéb elérési hibák (pl. nincs jogosultság)
	Bytes    int64 // megtalált fájlok össz. mérete
	Err      string
}

func (a *analysis) Total() int { return len(a.Entries) }

// Beolvassa a listát és minden bejegyzésről megállapítja, megvan-e a fájl. Nem másol semmit.
func analyzeList(listPath string, enc int) *analysis {
	a := &analysis{Path: listPath}
	data, err := os.ReadFile(listPath)
	if err != nil {
		a.Err = err.Error()
		return a
	}
	listDir := filepath.Dir(listPath)
	entries, label := loadEntries(data, listDir, enc)
	a.Enc = label
	a.Entries = make([]entryInfo, 0, len(entries))
	for _, e := range entries {
		ei := entryInfo{Raw: e}
		p, local := resolveEntry(e, listDir)
		if !local {
			ei.Path, ei.Status = e, stNotLocal
			a.NotLocal++
			a.Entries = append(a.Entries, ei)
			continue
		}
		ei.Path = p
		st, err := os.Stat(p)
		switch {
		case err == nil && st.IsDir():
			ei.Status = stDir
			a.Dirs++
		case err == nil:
			ei.Status, ei.Size = stOK, st.Size()
			a.Found++
			a.Bytes += st.Size()
		case errors.Is(err, os.ErrNotExist):
			ei.Status = stMissing
			a.NotFound++
		default:
			ei.Status = stError
			a.Errs++
		}
		a.Entries = append(a.Entries, ei)
	}
	return a
}

func statusText(st int) string {
	switch st {
	case stOK:
		return "OK"
	case stMissing:
		return "HIÁNYZIK"
	case stNotLocal:
		return "NEM HELYI"
	case stDir:
		return "MAPPA"
	}
	return "HIBA"
}

func humanSize(n int64) string {
	const k = 1024.0
	f := float64(n)
	var s string
	switch {
	case f >= k*k*k:
		s = fmt.Sprintf("%.2f GB", f/(k*k*k))
	case f >= k*k:
		s = fmt.Sprintf("%.1f MB", f/(k*k))
	case f >= k:
		s = fmt.Sprintf("%.0f KB", f/k)
	default:
		s = fmt.Sprintf("%d B", n)
	}
	return strings.Replace(s, ".", ",", 1)
}

// Egy lista összegzése egy sorban.
func (a *analysis) Summary() string {
	if a.Err != "" {
		return "a lista nem olvasható: " + a.Err
	}
	s := fmt.Sprintf("%d bejegyzés, ebből beolvasva (megtalálva): %d", a.Total(), a.Found)
	if a.NotFound > 0 {
		s += fmt.Sprintf(", hiányzik: %d", a.NotFound)
	}
	if a.NotLocal > 0 {
		s += fmt.Sprintf(", nem helyi (pl. stream): %d", a.NotLocal)
	}
	if a.Dirs > 0 {
		s += fmt.Sprintf(", mappa: %d", a.Dirs)
	}
	if a.Errs > 0 {
		s += fmt.Sprintf(", elérési hiba: %d", a.Errs)
	}
	if a.Found > 0 {
		s += " (" + humanSize(a.Bytes) + ")"
	}
	return s + " – kódolás: " + a.Enc
}
