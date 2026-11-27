package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

// Kódolás-választók (a legördülő menü sorrendje)
const (
	encAuto = iota
	encUTF8
	enc1250
	enc852
	enc1252
)

var schemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.\-]+://`)

func codePageOf(enc int) int {
	switch enc {
	case enc852:
		return 852
	case enc1252:
		return 1252
	default:
		return 1250
	}
}

func encName(enc int) string {
	switch enc {
	case encUTF8:
		return "UTF-8"
	case enc852:
		return "DOS 852"
	case enc1252:
		return "Windows-1252"
	default:
		return "Windows-1250"
	}
}

func decodeUTF16(b []byte, bigEndian bool) string {
	n := len(b) / 2
	u := make([]uint16, n)
	for i := 0; i < n; i++ {
		if bigEndian {
			u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		} else {
			u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
		}
	}
	return string(utf16.Decode(u))
}

// BOM alapján eldönthető kódolás (UTF-8 / UTF-16).
func decodeBOM(b []byte) (string, bool) {
	switch {
	case len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF:
		return string(b[3:]), true
	case len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE:
		return decodeUTF16(b[2:], false), true
	case len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF:
		return decodeUTF16(b[2:], true), true
	}
	return "", false
}

// A lista sorainak feldolgozása: üres és # kezdetű sorok kihagyva.
func parseEntries(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var out []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(strings.TrimPrefix(l, "\uFEFF"))
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		out = append(out, l)
	}
	return out
}

// Egy bejegyzésből teljes helyi útvonal. A második érték false, ha nem helyi fájl (pl. http stream).
func resolveEntry(entry, listDir string) (string, bool) {
	p := strings.Trim(strings.TrimSpace(entry), `"`)

	if len(p) >= 7 && strings.EqualFold(p[:7], "file://") {
		rest := p[7:]
		if un, err := url.PathUnescape(rest); err == nil {
			rest = un
		}
		if len(rest) >= 10 && strings.EqualFold(rest[:10], "localhost/") {
			rest = rest[9:]
		}
		switch {
		case len(rest) >= 3 && rest[0] == '/' && rest[2] == ':':
			rest = rest[1:] // /C:/zene -> C:/zene
		case strings.HasPrefix(rest, "/"):
			// marad
		default:
			rest = "//" + rest // file://szerver/megosztas
		}
		p = rest
	} else if schemeRe.MatchString(p) {
		return "", false
	}

	p = filepath.FromSlash(p)

	// \Zene\x.mp3: meghajtó nélküli útvonal -> a lista meghajtójára értjük
	if vol := filepath.VolumeName(listDir); vol != "" {
		if strings.HasPrefix(p, `\`) && !strings.HasPrefix(p, `\\`) {
			p = vol + p
		}
	}

	if !filepath.IsAbs(p) {
		p = filepath.Join(listDir, p)
	}
	return filepath.Clean(p), true
}

func countExisting(entries []string, listDir string) int {
	n := 0
	for _, e := range entries {
		if p, ok := resolveEntry(e, listDir); ok {
			if _, err := os.Stat(p); err == nil {
				n++
			}
		}
	}
	return n
}

// Kiválasztja a lista kódolását, és visszaadja a sorokat.
func loadEntries(b []byte, listDir string, enc int) ([]string, string) {
	if text, ok := decodeBOM(b); ok {
		return parseEntries(text), "UTF-8/UTF-16 (BOM alapján)"
	}
	switch enc {
	case encUTF8:
		return parseEntries(strings.ToValidUTF8(string(b), "\uFFFD")), "UTF-8"
	case enc1250, enc852, enc1252:
		return parseEntries(decodeANSI(codePageOf(enc), b)), encName(enc)
	}
	// Automatikus. Tiszta ASCII esetén nincs mit találgatni. Egyébként a lehetséges kódolások
	// közül azt választjuk, amelyikkel a legtöbb fájl valóban megtalálható. Egy régi ANSI lista
	// bájtjai véletlenül lehetnek érvényes UTF-8-ak, ezért az érvényes UTF-8 sem elég bizonyíték.
	type cand struct {
		entries []string
		label   string
		n       int
	}
	var cands []cand
	if isASCII(b) {
		return parseEntries(string(b)), "ASCII"
	}
	if utf8.Valid(b) {
		entries := parseEntries(string(b))
		n := countExisting(entries, listDir)
		if n == len(entries) {
			return entries, "UTF-8" // minden fájl megvan, biztos a találat
		}
		cands = append(cands, cand{entries, "UTF-8", n})
	}
	for _, e := range []int{enc1250, enc852, enc1252} {
		entries := parseEntries(decodeANSI(codePageOf(e), b))
		cands = append(cands, cand{entries, encName(e), countExisting(entries, listDir)})
	}
	best := cands[0] // döntetlennél az első (UTF-8, ha érvényes, különben Windows-1250)
	for _, c := range cands[1:] {
		if c.n > best.n {
			best = c
		}
	}
	return best.entries, best.label + " (automatikusan felismerve)"
}

func isASCII(b []byte) bool {
	for _, c := range b {
		if c >= 0x80 {
			return false
		}
	}
	return true
}

type counters struct {
	Copied, Existed, Missing, Skipped, Errors int // Skipped: nem helyi bejegyzés vagy mappa
}

type job struct {
	Enc      int
	Sub      bool
	Log      func(string)
	Progress func(done, total int)
	Cancel   *atomic.Bool
	Verify   bool // másolás után a tartalom bájtonkénti összehasonlítása
}

const maxSuffix = 9999

// Két fájl tartalmának bájtonkénti összehasonlítása. A méret csak gyors előszűrés.
func sameContent(a, b string) (bool, error) {
	ia, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	ib, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if !ia.Mode().IsRegular() || !ib.Mode().IsRegular() {
		return false, nil
	}
	if os.SameFile(ia, ib) {
		return true, nil
	}
	if ia.Size() != ib.Size() {
		return false, nil
	}
	fa, err := os.Open(a)
	if err != nil {
		return false, err
	}
	defer fa.Close()
	fb, err := os.Open(b)
	if err != nil {
		return false, err
	}
	defer fb.Close()

	isEnd := func(e error) bool { return e == io.EOF || e == io.ErrUnexpectedEOF }
	ba, bb := make([]byte, 1<<20), make([]byte, 1<<20)
	for {
		na, ea := io.ReadFull(fa, ba)
		nb, eb := io.ReadFull(fb, bb)
		if ea != nil && !isEnd(ea) {
			return false, ea
		}
		if eb != nil && !isEnd(eb) {
			return false, eb
		}
		if na != nb || !bytes.Equal(ba[:na], bb[:nb]) {
			return false, nil
		}
		if ea != nil || eb != nil { // legalább az egyik véget ért, és eddig azonosak
			return ea != nil && eb != nil, nil
		}
	}
}

const (
	actCopy   = iota // új célfájl készül
	actExists        // ugyanez a tartalom már a célmappában van
)

// Megkeresi, hová kerüljön a fájl: az eredeti néven; ha ott más tartalmú fájl van,
// "név (2)", "név (3)" stb. néven. Ha valamelyik változat tartalma azonos a forrással,
// nem másolunk újra (így az ismételt futtatás nem halmoz fel másolatokat).
func resolveTarget(dir, name, src string) (string, int, error) {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i <= maxSuffix; i++ {
		cand := name
		if i > 1 {
			cand = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		p := filepath.Join(dir, cand)
		_, err := os.Lstat(p)
		if errors.Is(err, os.ErrNotExist) {
			return p, actCopy, nil
		}
		if err != nil {
			return "", 0, err // pl. jogosultsági hiba: nem próbálkozunk a végtelenségig
		}
		same, err := sameContent(src, p)
		if err != nil {
			return "", 0, err
		}
		if same {
			return p, actExists, nil
		}
	}
	return "", 0, fmt.Errorf("túl sok azonos nevű fájl a célmappában: %s", name)
}

func copyFile(src, dst string, srcInfo os.FileInfo, verify bool) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, in)
	if err == nil && n != srcInfo.Size() {
		err = fmt.Errorf("a másolt méret (%d) eltér a forrásétól (%d) – a forrás közben megváltozott?", n, srcInfo.Size())
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil { // a célfájl tényleges mérete
		if st, e := os.Stat(dst); e != nil {
			err = e
		} else if st.Size() != n {
			err = fmt.Errorf("a célfájl mérete (%d) nem egyezik a másolt adatéval (%d)", st.Size(), n)
		}
	}
	if err == nil && verify {
		same, e := sameContent(src, dst)
		if e != nil {
			err = e
		} else if !same {
			err = errors.New("ellenőrzés sikertelen: a másolat tartalma eltér a forrásétól")
		}
	}
	if err != nil {
		os.Remove(dst) // ne maradjon félkész fájl
		return err
	}
	_ = os.Chtimes(dst, time.Now(), srcInfo.ModTime())
	return nil
}

func (j *job) run(listPath, destRoot string, c *counters) {
	j.Log("")
	j.Log("=== " + listPath)

	data, err := os.ReadFile(listPath)
	if err != nil {
		j.Log("HIBA: a lista nem olvasható: " + err.Error())
		c.Errors++
		return
	}
	listDir := filepath.Dir(listPath)
	entries, encLabel := loadEntries(data, listDir, j.Enc)
	j.Log(fmt.Sprintf("Kódolás: %s, %d bejegyzés.", encLabel, len(entries)))

	destDir := destRoot
	if j.Sub {
		base := filepath.Base(listPath)
		destDir = filepath.Join(destRoot, strings.TrimSuffix(base, filepath.Ext(base)))
	}
	if err := os.MkdirAll(destDir, 0o777); err != nil {
		j.Log("HIBA: a célmappa nem hozható létre: " + err.Error())
		c.Errors++
		return
	}

	total := len(entries)
	for i, entry := range entries {
		if j.Cancel != nil && j.Cancel.Load() {
			j.Log("MEGSZAKÍTVA.")
			return
		}
		j.Progress(i+1, total)

		src, local := resolveEntry(entry, listDir)
		if !local {
			j.Log("KIHAGYVA (nem helyi fájl): " + entry)
			c.Skipped++
			continue
		}
		info, err := os.Stat(src)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				j.Log("NEM TALÁLHATÓ: " + src)
				c.Missing++
			} else {
				j.Log("HIBA: " + src + " -> " + err.Error())
				c.Errors++
			}
			continue
		}
		if info.IsDir() {
			j.Log("KIHAGYVA (mappa): " + src)
			c.Skipped++
			continue
		}

		target, act, err := resolveTarget(destDir, filepath.Base(src), src)
		if err != nil {
			j.Log("HIBA: " + src + " -> " + err.Error())
			c.Errors++
			continue
		}
		name := filepath.Base(target)
		if act == actExists {
			j.Log("MEGVAN (azonos tartalom): " + name)
			c.Existed++
			continue
		}
		if err := copyFile(src, target, info, j.Verify); err != nil {
			j.Log("HIBA: " + src + " -> " + err.Error())
			c.Errors++
			continue
		}
		c.Copied++
		note := ""
		if j.Verify {
			note = " (ellenőrizve)"
		}
		j.Log(fmt.Sprintf("OK  [%d/%d] %s%s", i+1, total, name, note))
	}
}
