package press

import (
	"io"
	"time"
)

type Post struct {
	SourcePath string
	SourceDir  string

	Slug  string
	URL   string
	Title string
	Date  time.Time

	Body HTML
}

type HTML interface {
	Write(io.Writer) error
}
