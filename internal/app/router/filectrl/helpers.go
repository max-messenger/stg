package filectrl

import (
	"strings"
	"sync"
)

var copyBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 32*1024)

		return &buf
	},
}

func sanitizeFilename(name string) string {
	replaced := strings.NewReplacer(
		"\"", "'",
		"\n", "",
		"\r", "",
	)

	return replaced.Replace(name)
}
