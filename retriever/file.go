package retriever

import "io"

type File struct {
	Name   string
	Reader io.Reader
}
