package memorymanager

import "io"

type File struct {
	Name   string
	Reader io.Reader
}
