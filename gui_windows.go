//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// A -ldflags "-X main.version=..." felülírja.
var version = "1.2.0"

// ---------------------------------------------------------------- Win32 betöltés

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	pRegisterClassExW    = user32.NewProc("RegisterClassExW")
	pCreateWindowExW     = user32.NewProc("CreateWindowExW")
	pDefWindowProcW      = user32.NewProc("DefWindowProcW")
	pGetMessageW         = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessageW    = user32.NewProc("DispatchMessageW")
	pIsDialogMessageW    = user32.NewProc("IsDialogMessageW")
	pPostQuitMessage     = user32.NewProc("PostQuitMessage")
	pShowWindow          = user32.NewProc("ShowWindow")
	pUpdateWindow        = user32.NewProc("UpdateWindow")
	pSendMessageW        = user32.NewProc("SendMessageW")
	pPostMessageW        = user32.NewProc("PostMessageW")
	pMoveWindow          = user32.NewProc("MoveWindow")
	pGetClientRect       = user32.NewProc("GetClientRect")
	pGetWindowTextW      = user32.NewProc("GetWindowTextW")
	pGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	pSetWindowTextW      = user32.NewProc("SetWindowTextW")
	pEnableWindow        = user32.NewProc("EnableWindow")
	pMessageBoxW         = user32.NewProc("MessageBoxW")
	pLoadCursorW         = user32.NewProc("LoadCursorW")
	pLoadIconW           = user32.NewProc("LoadIconW")
	pGetSysColorBrush    = user32.NewProc("GetSysColorBrush")
	pGetDpiForSystem     = user32.NewProc("GetDpiForSystem")
	pGetDC               = user32.NewProc("GetDC")
	pReleaseDC           = user32.NewProc("ReleaseDC")
	pSetFocus            = user32.NewProc("SetFocus")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	pCreateFontW   = gdi32.NewProc("CreateFontW")
	pGetDeviceCaps = gdi32.NewProc("GetDeviceCaps")

	pInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")

	pGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")

	pDragAcceptFiles      = shell32.NewProc("DragAcceptFiles")
	pDragQueryFileW       = shell32.NewProc("DragQueryFileW")
	pDragFinish           = shell32.NewProc("DragFinish")
	pSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")

	pOleInitialize = ole32.NewProc("OleInitialize")
	pCoTaskMemFree = ole32.NewProc("CoTaskMemFree")
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsTabStop          = 0x00010000
	wsVScroll          = 0x00200000
	wsHScroll          = 0x00100000
	wsExClientEdge     = 0x00000200
	wsExAcceptFiles    = 0x00000010

	esMultiline   = 0x0004
	esAutoVScroll = 0x0040
	esAutoHScroll = 0x0080
	esReadOnly    = 0x0800

	bsPushButton   = 0x0
	bsAutoCheckbox = 0x3

	lbsNotify         = 0x0001
	lbsExtendedSel    = 0x0800
	lbsNoIntegralH    = 0x0100
	cbsDropDownList   = 0x0003
	swShow            = 5
	idcArrow          = 32512
	colorBtnFace      = 15
	cwUseDefault      = 0x80000000
	wmCreate          = 0x0001
	wmDestroy         = 0x0002
	wmSize            = 0x0005
	wmClose           = 0x0010
	wmGetMinMaxInfo   = 0x0024
	wmSetFont         = 0x0030
	wmCommand         = 0x0111
	wmDropFiles       = 0x0233
	wmApp             = 0x8000
	wmLogFlush        = wmApp + 1
	wmProgress        = wmApp + 2
	wmWorkDone        = wmApp + 3
	wmGetTextLength   = 0x000E
	lbAddString       = 0x0180
	lbDeleteString    = 0x0182
	lbGetCount        = 0x018B
	lbGetText         = 0x0189
	lbGetTextLen      = 0x018A
	lbGetSelCount     = 0x0190
	lbGetSelItems     = 0x0191
	lbSetHExtent      = 0x0194
	lbFindStringExact = 0x01A2
	cbAddString       = 0x0143
	cbGetCurSel       = 0x0147
	cbSetCurSel       = 0x014E
	bmGetCheck        = 0x00F0
	emSetSel          = 0x00B1
	emReplaceSel      = 0x00C2
	emLimitText       = 0x00C5
	emScrollCaret     = 0x00B7
	pbmSetPos         = 0x0402
	pbmSetRange32     = 0x0406

	ofnAllowMultiSelect = 0x00000200
	ofnExplorer         = 0x00080000
	ofnFileMustExist    = 0x00001000
	ofnPathMustExist    = 0x00000800
	ofnNoChangeDir      = 0x00000008

	mbOK          = 0x0
	mbIconWarning = 0x30
	mbIconInfo    = 0x40

	idList     = 101
	idAdd      = 102
	idRemove   = 103
	idDest     = 104
	idBrowse   = 105
	idEnc      = 106
	idSub      = 107
	idStart    = 108
	idProgress = 109
	idLog      = 110
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type point struct{ x, y int32 }

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type rect struct{ left, top, right, bottom int32 }

type minMaxInfo struct {
	reserved, maxSize, maxPosition, minTrack, maxTrack point
}

type initCommonControlsEx struct{ size, icc uint32 }

type openFileName struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

type browseInfo struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

// ---------------------------------------------------------------- segédek

func wstr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(strings.ReplaceAll(s, "\x00", ""))
	return p
}

func call(p *syscall.LazyProc, a ...uintptr) uintptr {
	r, _, _ := p.Call(a...)
	return r
}

func sendMsg(h uintptr, m uint32, w, l uintptr) uintptr {
	return call(pSendMessageW, h, uintptr(m), w, l)
}

func sendMsgStr(h uintptr, m uint32, w uintptr, s string) uintptr {
	p := wstr(s)
	r := call(pSendMessageW, h, uintptr(m), w, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
	return r
}

func i32(v int32) uintptr { return uintptr(v) }

func getText(h uintptr) string {
	n := int(call(pGetWindowTextLength, h))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	call(pGetWindowTextW, h, uintptr(unsafe.Pointer(&buf[0])), uintptr(n+1))
	return syscall.UTF16ToString(buf)
}

func setText(h uintptr, s string) {
	p := wstr(s)
	call(pSetWindowTextW, h, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
}

func messageBox(text, title string, flags uintptr) {
	t, c := wstr(text), wstr(title)
	call(pMessageBoxW, hMain, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
	runtime.KeepAlive(t)
	runtime.KeepAlive(c)
}

// ---------------------------------------------------------------- állapot

var (
	hInst                                 uintptr
	hMain                                 uintptr
	hLblList, hLblDest, hLblEnc           uintptr
	hList, hBtnAdd, hBtnRemove            uintptr
	hDest, hBtnBrowse, hEnc, hSub, hStart uintptr
	hProg, hLog, hVerify                  uintptr
	hFont, hMono                          uintptr
	dpi                                   = 96
	running                               atomic.Bool
	cancelFlag                            atomic.Bool
	uiMu                                  sync.Mutex
	uiQueue                               []string
	wndProcCB                             = syscall.NewCallback(wndProc)
)

func sc(v int) int32 { return int32(v * dpi / 96) }

// ---------------------------------------------------------------- lista tartalma (elemzés)

const (
	idOnlyMissing = 111
	idContent     = 112
	idVerify      = 113

	wmAnalyzed     = wmApp + 4
	wmSetRedraw    = 0x000B
	lbResetContent = 0x0184
	lbSetSel       = 0x0185
	lbInitStorage  = 0x01A8
	lbsNoSel       = 0x4000
	ssNoPrefix     = 0x0080
	lbnSelChange   = 1
	cbnSelChange   = 1
)

var (
	hSummary, hOnlyMissing, hContent uintptr
	pGetSystemMetrics                = user32.NewProc("GetSystemMetrics")

	// uiMu védi:
	analyses    = map[string]*analysis{}
	pendingJobs = map[string]bool{}
	analysisGen int
	logSaveErr  string // a worker állítja, a felület jelzi (uiMu védi)
)

func minH() int32 {
	h := sc(600)
	if s := int32(call(pGetSystemMetrics, 1)); s > 0 && h > s-sc(40) {
		h = s - sc(40)
	}
	return h
}

func currentEnc() int {
	e := int(sendMsg(hEnc, cbGetCurSel, 0, 0))
	if e < 0 {
		e = 0
	}
	return e
}

// Háttérszálon beolvassa a listát és megnézi, mely fájlok léteznek.
func startAnalysis(path string) {
	uiMu.Lock()
	gen := analysisGen
	delete(analyses, path)
	pendingJobs[path] = true
	uiMu.Unlock()
	enc := currentEnc()
	go func() {
		a := analyzeList(path, enc)
		uiMu.Lock()
		if gen == analysisGen && pendingJobs[path] {
			analyses[path] = a
			delete(pendingJobs, path)
		}
		uiMu.Unlock()
		call(pPostMessageW, hMain, wmAnalyzed, 0, 0)
	}()
}

func restartAllAnalyses() {
	uiMu.Lock()
	analysisGen++
	analyses = map[string]*analysis{}
	pendingJobs = map[string]bool{}
	uiMu.Unlock()
	for _, l := range getLists() {
		startAnalysis(l)
	}
}

func pruneAnalyses() {
	keep := map[string]bool{}
	for _, l := range getLists() {
		keep[l] = true
	}
	uiMu.Lock()
	for k := range analyses {
		if !keep[k] {
			delete(analyses, k)
		}
	}
	for k := range pendingJobs {
		if !keep[k] {
			delete(pendingJobs, k)
		}
	}
	uiMu.Unlock()
}

// A kijelölt (ennek hiányában az utolsó) lista indexe, -1 ha nincs lista.
func shownListIndex() int {
	cnt := int(sendMsg(hList, lbGetCount, 0, 0))
	if cnt <= 0 {
		return -1
	}
	if n := int(sendMsg(hList, lbGetSelCount, 0, 0)); n > 0 {
		items := make([]int32, n)
		sendMsg(hList, lbGetSelItems, uintptr(n), uintptr(unsafe.Pointer(&items[0])))
		return int(items[0])
	}
	return cnt - 1
}

// Frissíti az összegzést és a tartalomlistát.
func refreshView() {
	lists := getLists()
	idx := shownListIndex()

	uiMu.Lock()
	var shown *analysis
	shownPending := false
	if idx >= 0 && idx < len(lists) {
		shown = analyses[lists[idx]]
		shownPending = pendingJobs[lists[idx]]
	}
	var nTotal, nFound, nNotFound, nNotLocal, nDirs, nErrs, nAnalyzed, nPending int
	var bytes int64
	for _, l := range lists {
		if a := analyses[l]; a != nil {
			nAnalyzed++
			nTotal += a.Total()
			nFound += a.Found
			nNotFound += a.NotFound
			nNotLocal += a.NotLocal
			nDirs += a.Dirs
			nErrs += a.Errs
			bytes += a.Bytes
		} else if pendingJobs[l] {
			nPending++
		}
	}
	uiMu.Unlock()

	// összegzés
	var line1, line2 string
	switch {
	case idx < 0 || idx >= len(lists):
		line1 = "Még nincs lista hozzáadva – húzz be egy .m3u / .m3u8 fájlt."
	case shown != nil:
		line1 = filepath.Base(lists[idx]) + ": " + shown.Summary()
	case shownPending:
		line1 = filepath.Base(lists[idx]) + ": beolvasás folyamatban..."
	default:
		line1 = filepath.Base(lists[idx])
	}
	if len(lists) > 0 {
		line2 = fmt.Sprintf("Összesen: %d lista, %d bejegyzés, megtalálva: %d, hiányzik: %d",
			len(lists), nTotal, nFound, nNotFound)
		if nNotLocal > 0 {
			line2 += fmt.Sprintf(", nem helyi: %d", nNotLocal)
		}
		if nDirs > 0 {
			line2 += fmt.Sprintf(", mappa: %d", nDirs)
		}
		if nErrs > 0 {
			line2 += fmt.Sprintf(", elérési hiba: %d", nErrs)
		}
		if nFound > 0 {
			line2 += " (" + humanSize(bytes) + ")"
		}
		if nPending > 0 {
			line2 += fmt.Sprintf(" – még beolvasás alatt: %d lista", nPending)
		}
	}
	setText(hSummary, line1+"\r\n"+line2)

	// tartalom
	onlyMissing := sendMsg(hOnlyMissing, bmGetCheck, 0, 0) == 1
	var rows []string
	switch {
	case shown == nil:
		if shownPending {
			rows = []string{"(beolvasás folyamatban...)"}
		}
	case shown.Err != "":
		rows = []string{"HIBA: " + shown.Err}
	default:
		for _, e := range shown.Entries {
			if onlyMissing && e.Status == stOK {
				continue
			}
			size := ""
			if e.Status == stOK {
				size = humanSize(e.Size)
			}
			rows = append(rows, fmt.Sprintf("%-10s%10s  %s", statusText(e.Status), size, e.Path))
		}
		if len(rows) == 0 {
			if onlyMissing && shown.Total() > 0 {
				rows = []string{"(nincs hiányzó fájl – mind megvan)"}
			} else {
				rows = []string{"(a listában nincs bejegyzés)"}
			}
		}
	}

	sendMsg(hContent, wmSetRedraw, 0, 0)
	sendMsg(hContent, lbResetContent, 0, 0)
	sendMsg(hContent, lbInitStorage, uintptr(len(rows)), uintptr(len(rows)*120))
	maxLen := 0
	for _, r := range rows {
		sendMsgStr(hContent, lbAddString, 0, r)
		if n := len([]rune(r)); n > maxLen {
			maxLen = n
		}
	}
	sendMsg(hContent, lbSetHExtent, uintptr(int(sc(8))*maxLen+int(sc(30))), 0)
	sendMsg(hContent, wmSetRedraw, 1, 0)
	call(pInvalidateRect, hContent, 0, 1)
}

var pInvalidateRect = user32.NewProc("InvalidateRect")

func createCtl(class, text string, style, exStyle uint32, id int) uintptr {
	c, t := wstr(class), wstr(text)
	h := call(pCreateWindowExW,
		uintptr(exStyle), uintptr(unsafe.Pointer(c)), uintptr(unsafe.Pointer(t)),
		uintptr(style|wsChild|wsVisible),
		0, 0, 10, 10, hMain, uintptr(id), hInst, 0)
	runtime.KeepAlive(c)
	runtime.KeepAlive(t)
	return h
}

func place(h uintptr, x, y, w, hgt int32) {
	call(pMoveWindow, h, i32(x), i32(y), i32(w), i32(hgt), 1)
}

func layout() {
	if hList == 0 || hMain == 0 {
		return
	}
	var rc rect
	call(pGetClientRect, hMain, uintptr(unsafe.Pointer(&rc)))
	W := rc.right - rc.left
	H := rc.bottom - rc.top
	m := sc(12)
	btnW := sc(130)
	inW := W - 2*m - btnW - sc(8)

	place(hLblList, m, sc(12), W-2*m, sc(18))
	place(hList, m, sc(34), inW, sc(84))
	place(hBtnAdd, W-m-btnW, sc(34), btnW, sc(30))
	place(hBtnRemove, W-m-btnW, sc(70), btnW, sc(30))

	place(hSummary, m, sc(124), W-2*m, sc(50))

	place(hLblDest, m, sc(178), W-2*m, sc(18))
	place(hDest, m, sc(198), inW, sc(26))
	place(hBtnBrowse, W-m-btnW, sc(196), btnW, sc(30))

	place(hLblEnc, m, sc(238), sc(110), sc(18))
	place(hEnc, m+sc(112), sc(234), sc(280), sc(200))
	place(hSub, m, sc(266), sc(280), sc(22))
	place(hVerify, m+sc(292), sc(266), W-2*m-sc(292), sc(22))

	place(hStart, m, sc(298), sc(170), sc(34))
	place(hProg, m+sc(182), sc(305), W-2*m-sc(182), sc(20))

	// a maradék helyen osztozik a lista tartalma és a napló
	place(hOnlyMissing, m, sc(342), W-2*m, sc(22))
	areaTop := sc(368)
	avail := H - areaTop - m
	if avail < sc(120) {
		avail = sc(120)
	}
	ch := avail * 55 / 100
	place(hContent, m, areaTop, W-2*m, ch)
	place(hLog, m, areaTop+ch+sc(8), W-2*m, avail-ch-sc(8))
}

func addList(path string) {
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		return
	}
	p := wstr(path)
	idx := int32(call(pSendMessageW, hList, lbFindStringExact, i32(-1), uintptr(unsafe.Pointer(p))))
	isNew := idx == -1
	if isNew {
		idx = int32(call(pSendMessageW, hList, lbAddString, 0, uintptr(unsafe.Pointer(p))))
	}
	runtime.KeepAlive(p)
	if idx < 0 {
		return
	}
	// az újonnan hozzáadott (vagy már meglévő) lista legyen a kijelölt, annak mutatjuk a tartalmát
	sendMsg(hList, lbSetSel, 0, i32(-1))
	sendMsg(hList, lbSetSel, 1, uintptr(idx))
	if isNew {
		startAnalysis(path)
	}
	refreshView()
}

func getLists() []string {
	n := int(sendMsg(hList, lbGetCount, 0, 0))
	var out []string
	for i := 0; i < n; i++ {
		l := int(sendMsg(hList, lbGetTextLen, uintptr(i), 0))
		if l <= 0 {
			continue
		}
		buf := make([]uint16, l+1)
		sendMsg(hList, lbGetText, uintptr(i), uintptr(unsafe.Pointer(&buf[0])))
		out = append(out, syscall.UTF16ToString(buf))
	}
	return out
}

func appendLog(text string) {
	n := sendMsg(hLog, wmGetTextLength, 0, 0)
	sendMsg(hLog, emSetSel, n, n)
	sendMsgStr(hLog, emReplaceSel, 0, text)
	sendMsg(hLog, emScrollCaret, 0, 0)
}

func postLog(s string) {
	uiMu.Lock()
	uiQueue = append(uiQueue, s)
	uiMu.Unlock()
	call(pPostMessageW, hMain, wmLogFlush, 0, 0)
}

func flushLog() {
	uiMu.Lock()
	q := uiQueue
	uiQueue = nil
	uiMu.Unlock()
	if len(q) > 0 {
		appendLog(strings.Join(q, "\r\n") + "\r\n")
	}
}

func setRunningUI(on bool) {
	en := uintptr(1)
	if on {
		en = 0
	}
	for _, h := range []uintptr{hList, hBtnAdd, hBtnRemove, hDest, hBtnBrowse, hEnc, hSub, hVerify} {
		call(pEnableWindow, h, en)
	}
	if on {
		setText(hStart, "Megszakítás")
	} else {
		setText(hStart, "Másolás indítása")
	}
}

// ---------------------------------------------------------------- párbeszédablakok

func pickFiles() []string {
	buf := make([]uint16, 65536)
	filter := utf16.Encode([]rune("Lejátszási listák (*.m3u;*.m3u8)\x00*.m3u;*.m3u8\x00Minden fájl (*.*)\x00*.*\x00"))
	filter = append(filter, 0)
	title := wstr("Válaszd ki a lejátszási listá(ka)t")
	ofn := openFileName{
		hwndOwner:    hMain,
		lpstrFilter:  &filter[0],
		nFilterIndex: 1,
		lpstrFile:    &buf[0],
		nMaxFile:     uint32(len(buf)),
		lpstrTitle:   title,
		flags:        ofnAllowMultiSelect | ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnNoChangeDir,
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	r := call(pGetOpenFileNameW, uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(filter)
	runtime.KeepAlive(title)
	if r == 0 {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] == 0 {
			if i == start {
				break
			}
			parts = append(parts, syscall.UTF16ToString(buf[start:i]))
			start = i + 1
		}
	}
	if len(parts) <= 1 {
		return parts
	}
	var out []string
	for _, name := range parts[1:] {
		out = append(out, filepath.Join(parts[0], name))
	}
	return out
}

func pickFolder() string {
	name := make([]uint16, 1024)
	title := wstr("Válaszd ki a célmappát")
	bi := browseInfo{
		hwndOwner:      hMain,
		pszDisplayName: &name[0],
		lpszTitle:      title,
		ulFlags:        0x1 | 0x10 | 0x40, // csak fájlrendszer-mappa, szerkesztőmező, új stílus
	}
	pidl := call(pSHBrowseForFolderW, uintptr(unsafe.Pointer(&bi)))
	runtime.KeepAlive(title)
	if pidl == 0 {
		return ""
	}
	defer call(pCoTaskMemFree, pidl)
	path := make([]uint16, 1024)
	if call(pSHGetPathFromIDListW, pidl, uintptr(unsafe.Pointer(&path[0]))) == 0 {
		return ""
	}
	return syscall.UTF16ToString(path)
}

// ---------------------------------------------------------------- másolás indítása

func startWork() {
	lists := getLists()
	if len(lists) == 0 {
		messageBox("Adj hozzá legalább egy lejátszási listát.", "Hiányzó lista", mbOK|mbIconWarning)
		return
	}
	dest := strings.TrimSpace(strings.Trim(strings.TrimSpace(getText(hDest)), `"`))
	if dest == "" {
		messageBox("Add meg a célmappát.", "Hiányzó célmappa", mbOK|mbIconWarning)
		return
	}
	enc := int(sendMsg(hEnc, cbGetCurSel, 0, 0))
	if enc < 0 {
		enc = 0
	}
	sub := sendMsg(hSub, bmGetCheck, 0, 0) == 1
	verify := sendMsg(hVerify, bmGetCheck, 0, 0) == 1

	setText(hLog, "")
	running.Store(true)
	cancelFlag.Store(false)
	setRunningUI(true)
	go worker(lists, dest, enc, sub, verify)
}

func worker(lists []string, dest string, enc int, sub, verify bool) {
	var lines []string
	logf := func(s string) {
		lines = append(lines, s)
		postLog(s)
	}
	defer call(pPostMessageW, hMain, wmWorkDone, 0, 0)
	defer func() {
		if r := recover(); r != nil {
			logf(fmt.Sprintf("LEÁLLT VÁRATLAN HIBA MIATT: %v – a másolás nem fejeződött be.", r))
		}
	}()

	if err := os.MkdirAll(dest, 0o777); err != nil {
		logf("LEÁLLT: a célmappa nem hozható létre: " + err.Error())
		return
	}

	var total counters
	j := &job{
		Enc: enc, Sub: sub, Verify: verify, Log: logf, Cancel: &cancelFlag,
		Progress: func(done, tot int) {
			call(pPostMessageW, hMain, wmProgress, uintptr(done), uintptr(tot))
		},
	}
	for _, l := range lists {
		if cancelFlag.Load() {
			break
		}
		j.run(l, dest, &total)
	}

	stats := fmt.Sprintf("Átmásolva: %d, már megvolt: %d, nem található: %d, kihagyva (nem helyi/mappa): %d, hiba: %d",
		total.Copied, total.Existed, total.Missing, total.Skipped, total.Errors)
	logf("")
	switch {
	case cancelFlag.Load():
		logf("MEGSZAKÍTVA – a másolás nem fejeződött be. " + stats)
	case total.Errors > 0:
		logf("BEFEJEZVE HIBÁKKAL. " + stats)
	case total.Missing > 0 || total.Skipped > 0:
		logf("BEFEJEZVE – de nem minden bejegyzés volt másolható. " + stats)
	default:
		logf("KÉSZ. " + stats)
	}

	logPath := filepath.Join(dest, "masolas_naplo.txt")
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte(strings.Join(lines, "\r\n")+"\r\n")...)
	if err := os.WriteFile(logPath, data, 0o666); err != nil {
		logf("FIGYELEM: a naplófájl nem menthető (" + logPath + "): " + err.Error())
		uiMu.Lock()
		logSaveErr = err.Error()
		uiMu.Unlock()
	} else {
		logf("Napló mentve: " + logPath)
	}
}

// ---------------------------------------------------------------- ablakkezelés

func onCommand(id, code int) {
	switch id {
	case idAdd:
		for _, f := range pickFiles() {
			addList(f)
		}
	case idRemove:
		n := int(sendMsg(hList, lbGetSelCount, 0, 0))
		if n <= 0 {
			return
		}
		items := make([]int32, n)
		sendMsg(hList, lbGetSelItems, uintptr(n), uintptr(unsafe.Pointer(&items[0])))
		for i := n - 1; i >= 0; i-- { // hátulról, hogy ne csússzanak el az indexek
			sendMsg(hList, lbDeleteString, uintptr(items[i]), 0)
		}
		pruneAnalyses()
		refreshView()
	case idList:
		if code == lbnSelChange {
			refreshView()
		}
	case idOnlyMissing:
		refreshView()
	case idEnc:
		if code == cbnSelChange {
			restartAllAnalyses() // más kódolással más fájlnevek jönnek ki
			refreshView()
		}
	case idBrowse:
		if p := pickFolder(); p != "" {
			setText(hDest, p)
		}
	case idStart:
		if running.Load() {
			cancelFlag.Store(true)
			postLog("Megszakítás kérve...")
		} else {
			startWork()
		}
	}
}

func onDrop(hdrop uintptr) {
	n := int(call(pDragQueryFileW, hdrop, 0xFFFFFFFF, 0, 0))
	for i := 0; i < n; i++ {
		l := int(call(pDragQueryFileW, hdrop, uintptr(i), 0, 0))
		buf := make([]uint16, l+1)
		call(pDragQueryFileW, hdrop, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), uintptr(l+1))
		addList(syscall.UTF16ToString(buf))
	}
	call(pDragFinish, hdrop)
}

func wndProc(hwnd, m, wParam, lParam uintptr) uintptr {
	if hMain == 0 {
		hMain = hwnd
	}
	switch uint32(m) {
	case wmSize:
		layout()
		return 0
	case wmGetMinMaxInfo:
		mm := (*minMaxInfo)(unsafe.Pointer(lParam))
		mm.minTrack = point{sc(640), minH()}
		return 0
	case wmAnalyzed:
		refreshView()
		return 0
	case wmCommand:
		onCommand(int(wParam&0xFFFF), int((wParam>>16)&0xFFFF))
		return 0
	case wmDropFiles:
		onDrop(wParam)
		return 0
	case wmLogFlush:
		flushLog()
		return 0
	case wmProgress:
		sendMsg(hProg, pbmSetRange32, 0, lParam)
		sendMsg(hProg, pbmSetPos, wParam, 0)
		return 0
	case wmWorkDone:
		flushLog()
		running.Store(false)
		setRunningUI(false)
		uiMu.Lock()
		lerr := logSaveErr
		logSaveErr = ""
		uiMu.Unlock()
		if lerr != "" {
			messageBox("A másolás lefutott, de a naplófájl (masolas_naplo.txt) nem menthető a célmappába:\r\n\r\n"+lerr+
				"\r\n\r\nA napló az ablakban továbbra is látható.", "A napló nem készült el", mbOK|mbIconWarning)
		}
		return 0
	case wmDestroy:
		call(pPostQuitMessage, 0)
		return 0
	}
	return call(pDefWindowProcW, hwnd, m, wParam, lParam)
}

func newFont(face string, pt int) uintptr {
	f := wstr(face)
	h := call(pCreateFontW,
		i32(-int32(pt*dpi/72)), 0, 0, 0, 400, 0, 0, 0,
		1,    // DEFAULT_CHARSET
		0, 0, // out/clip precision
		5, // CLEARTYPE_QUALITY
		0, uintptr(unsafe.Pointer(f)))
	runtime.KeepAlive(f)
	return h
}

func init() {
	runtime.LockOSThread() // az ablak üzenetciklusa ugyanazon az OS-szálon fusson
}

func main() {
	call(pOleInitialize, 0)
	icc := initCommonControlsEx{size: 8, icc: 0x20 | 0x4000} // progress + standard classes
	call(pInitCommonControlsEx, uintptr(unsafe.Pointer(&icc)))

	hInst = call(pGetModuleHandleW, 0)

	if pGetDpiForSystem.Find() == nil {
		if d := int(call(pGetDpiForSystem)); d > 0 {
			dpi = d
		}
	} else if dc := call(pGetDC, 0); dc != 0 {
		if d := int(call(pGetDeviceCaps, dc, 88)); d > 0 { // LOGPIXELSX
			dpi = d
		}
		call(pReleaseDC, 0, dc)
	}

	className := wstr("PlaylistMasoloWnd")
	wc := wndClassEx{
		lpfnWndProc:   wndProcCB,
		hInstance:     hInst,
		hIcon:         call(pLoadIconW, hInst, 1),
		hCursor:       call(pLoadCursorW, 0, idcArrow),
		hbrBackground: call(pGetSysColorBrush, colorBtnFace),
		lpszClassName: className,
		hIconSm:       call(pLoadIconW, hInst, 1),
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if call(pRegisterClassExW, uintptr(unsafe.Pointer(&wc))) == 0 {
		return
	}

	winH := sc(800)
	if s := int32(call(pGetSystemMetrics, 1)); s > 0 && winH > s-sc(60) {
		winH = s - sc(60) // kis képernyőn ne lógjon ki
	}
	title := wstr("Lejátszási lista másoló " + version)
	hwnd := call(pCreateWindowExW,
		wsExAcceptFiles, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow,
		cwUseDefault, cwUseDefault, uintptr(sc(820)), uintptr(winH),
		0, 0, hInst, 0)
	runtime.KeepAlive(className)
	runtime.KeepAlive(title)
	if hwnd == 0 {
		return
	}
	hMain = hwnd
	call(pDragAcceptFiles, hMain, 1)

	hFont = newFont("Segoe UI", 9)
	hMono = newFont("Consolas", 9)

	hLblList = createCtl("STATIC", "Lejátszási listák (.m3u / .m3u8) – ide húzhatod is őket:", 0, 0, 0)
	hList = createCtl("LISTBOX", "", wsTabStop|wsVScroll|wsHScroll|lbsNotify|lbsExtendedSel|lbsNoIntegralH, wsExClientEdge, idList)
	sendMsg(hList, lbSetHExtent, uintptr(sc(2000)), 0)
	hBtnAdd = createCtl("BUTTON", "Lista hozzáadása...", wsTabStop|bsPushButton, 0, idAdd)
	hBtnRemove = createCtl("BUTTON", "Kijelölt törlése", wsTabStop|bsPushButton, 0, idRemove)

	hLblDest = createCtl("STATIC", "Célmappa:", 0, 0, 0)
	hDest = createCtl("EDIT", "", wsTabStop|esAutoHScroll, wsExClientEdge, idDest)
	hBtnBrowse = createCtl("BUTTON", "Tallózás...", wsTabStop|bsPushButton, 0, idBrowse)

	hLblEnc = createCtl("STATIC", "Lista kódolása:", 0, 0, 0)
	hEnc = createCtl("COMBOBOX", "", wsTabStop|wsVScroll|cbsDropDownList, 0, idEnc)
	for _, s := range []string{
		"Automatikus (ajánlott)", "UTF-8", "Windows-1250 (magyar ANSI)",
		"DOS 852 (közép-európai)", "Windows-1252 (nyugat-európai)",
	} {
		sendMsgStr(hEnc, cbAddString, 0, s)
	}
	sendMsg(hEnc, cbSetCurSel, 0, 0)

	hSub = createCtl("BUTTON", "Külön almappa minden listának", wsTabStop|bsAutoCheckbox, 0, idSub)
	hVerify = createCtl("BUTTON", "Ellenőrzés másolás után (lassabb)", wsTabStop|bsAutoCheckbox, 0, idVerify)
	hStart = createCtl("BUTTON", "Másolás indítása", wsTabStop|bsPushButton, 0, idStart)
	hProg = createCtl("msctls_progress32", "", 0, 0, idProgress)
	hLog = createCtl("EDIT", "", wsTabStop|wsVScroll|esMultiline|esAutoVScroll|esReadOnly, wsExClientEdge, idLog)
	sendMsg(hLog, emLimitText, 0x10000000, 0)

	hSummary = createCtl("STATIC", "", ssNoPrefix, 0, 0) // SS_NOPREFIX: a fájlnevekben lévő & ne tűnjön el
	hOnlyMissing = createCtl("BUTTON", "Csak a hiányzó (nem található) fájlok mutatása a tartalomlistában", wsTabStop|bsAutoCheckbox, 0, idOnlyMissing)
	hContent = createCtl("LISTBOX", "", wsTabStop|wsVScroll|wsHScroll|lbsNoIntegralH|lbsNoSel, wsExClientEdge, idContent)

	for _, h := range []uintptr{hLblList, hList, hBtnAdd, hBtnRemove, hLblDest, hDest, hBtnBrowse,
		hLblEnc, hEnc, hSub, hVerify, hStart, hSummary, hOnlyMissing} {
		sendMsg(h, wmSetFont, hFont, 1)
	}
	sendMsg(hLog, wmSetFont, hMono, 1)
	sendMsg(hContent, wmSetFont, hMono, 1)

	layout()
	refreshView()
	call(pShowWindow, hMain, swShow)
	call(pUpdateWindow, hMain)

	var mm msg
	for {
		r := int32(call(pGetMessageW, uintptr(unsafe.Pointer(&mm)), 0, 0, 0))
		if r <= 0 {
			break
		}
		if call(pIsDialogMessageW, hMain, uintptr(unsafe.Pointer(&mm))) != 0 {
			continue
		}
		call(pTranslateMessage, uintptr(unsafe.Pointer(&mm)))
		call(pDispatchMessageW, uintptr(unsafe.Pointer(&mm)))
	}
}
