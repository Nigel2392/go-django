package fileutils

import (
	"iter"
	"mime"
	"strings"
)

func MimeTypesForExtsSeq(exts []string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, ext := range exts {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}

			var mimeType = mime.TypeByExtension(ext)
			if !yield(i, mimeType) {
				break
			}
		}
	}
}

func MimeTypesForExts(exts []string) []string {
	var mimeTypes = make([]string, 0, len(exts))
	for _, mimeType := range MimeTypesForExtsSeq(exts) {
		mimeTypes = append(mimeTypes, mimeType)
	}
	return mimeTypes
}
