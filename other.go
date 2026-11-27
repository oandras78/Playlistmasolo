//go:build !windows

package main

// Csak a Linuxon futó tesztekhez kell; a valódi program Windowsra készül.

func decodeANSI(cp int, b []byte) string {
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

func main() {}
