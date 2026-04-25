package press

import (
	"io"
)

type Renderers struct {
	Home      func(io.Writer, HomePageData) error
	BlogIndex func(io.Writer, BlogIndexPageData) error
	BlogPost  func(io.Writer, BlogPostPageData) error
}
