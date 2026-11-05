package domain

import "io"

type FileUpload struct {
	Key string
}

type FileDownload struct {
	Reader      io.ReadCloser
	ContentType string
	Size        int64
}
