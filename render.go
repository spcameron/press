package press

import (
	"io"
)

type Renderers struct {
	Home func(w io.Writer, data HomePageData) error
}
