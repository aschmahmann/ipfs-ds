// Note: code in this file almost exactly copied from the `ipfs cid bases` command in go-ipfs
package main

import (
	"fmt"
	"os"
	"sort"
	"unicode"

	"github.com/multiformats/go-multibase"
)

type CodeAndName struct {
	Code int
	Name string
}

type multibaseSorter struct {
	data []CodeAndName
}

func (s multibaseSorter) Len() int      { return len(s.data) }
func (s multibaseSorter) Swap(i, j int) { s.data[i], s.data[j] = s.data[j], s.data[i] }

func (s multibaseSorter) Less(i, j int) bool {
	a := unicode.ToLower(rune(s.data[i].Code))
	b := unicode.ToLower(rune(s.data[j].Code))
	if a != b {
		return a < b
	}
	// lowecase letters should come before uppercase
	return s.data[i].Code > s.data[j].Code
}

type codeAndNameSorter struct {
	data []CodeAndName
}

func (s codeAndNameSorter) Len() int           { return len(s.data) }
func (s codeAndNameSorter) Swap(i, j int)      { s.data[i], s.data[j] = s.data[j], s.data[i] }
func (s codeAndNameSorter) Less(i, j int) bool { return s.data[i].Code < s.data[j].Code }

func printBases(prefixes, numeric bool) {
	// write to standard out
	w := os.Stdout

	var res []CodeAndName
	// use EncodingToStr in case at some point there are multiple names for a given code
	for code, name := range multibase.EncodingToStr {
		res = append(res, CodeAndName{int(code), name})
	}

	sort.Sort(multibaseSorter{res})
	for _, v := range res {
		code := v.Code
		if code < 32 || code >= 127 {
			// don't display non-printable prefixes
			code = ' '
		}
		switch {
		case prefixes && numeric:
			fmt.Fprintf(w, "%c %5d  %s\n", code, v.Code, v.Name)
		case prefixes:
			fmt.Fprintf(w, "%c  %s\n", code, v.Name)
		case numeric:
			fmt.Fprintf(w, "%5d  %s\n", v.Code, v.Name)
		default:
			fmt.Fprintf(w, "%s\n", v.Name)
		}
	}
}
