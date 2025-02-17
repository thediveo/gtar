// Copyright 2025 by Harald Albrecht
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gtar

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"maps"
	"os"

	"golang.org/x/sys/unix"
)

type Index struct {
	idx map[string]seekableElement
	f   *os.File
}

// seekableElement describes an element inside a tar file so that the element's
// contents can later be seeked to and read on demand.
type seekableElement struct {
	*tar.Header
	Offset int64 // offset from beginning of file to contents.
}

// New returns a new TAR file Index object for the specified file path. Please
// note that the caller is responsible to
func New(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return NewFromFile(f)
}

// NewFromFile returns a new TAR file Index object for the specified os.File. As
// the Index returned maintains its own duplicated file descriptor, the caller
// is responsible to close the original os.File; this can be done immediately
// after the Index was created. When the Index is no longer needed, call
// [Index.Close] to release associated system resources.
func NewFromFile(tarf *os.File) (*Index, error) {
	// duplicate the caller's file to a new and independent os.File object, in
	// order to decouple the Index's file lifetime from the caller's os.File
	// object.
	newfd, err := unix.Dup(int(tarf.Fd()))
	if err != nil {
		return nil, err
	}
	index := &Index{
		idx: map[string]seekableElement{},
		f:   os.NewFile(uintptr(newfd), tarf.Name()),
	}

	junk := make([]byte, 4096)

	// Now run through the whole tape in order to build the index and record the
	// content offsets within the tar file.
	tarr := tar.NewReader(index.f)
	for {
		// Read the (next) header; if that succeeds, due to the (512 bytes)
		// block structure of the tar format, the file read position will be at
		// the beginning of the file contents.
		hdr, err := tarr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			index.Close()
			return nil, err
		}
		pos, err := index.f.Seek(0, io.SeekCurrent)
		if err != nil {
			index.Close()
			return nil, err
		}
		// Drain the contents, this should place the file read position
		// precisely beyond the last content byte, with (if any) padding still
		// pending. The padding will automatically be skipped when reading the
		// next header.
		for {
			_, err := tarr.Read(junk)
			if err == io.EOF {
				break
			}
			if err != nil {
				index.Close()
				return nil, err
			}
		}
		end, err := index.f.Seek(0, io.SeekCurrent)
		if err != nil {
			index.Close()
			return nil, err
		}
		if end != pos+hdr.Size {
			index.Close()
			return nil, fmt.Errorf("unsupported sparse file %q", hdr.Name)
		}
		index.idx[hdr.Name] = seekableElement{
			Header: hdr,
			Offset: pos,
		}
	}
	return index, nil
}

// Close the index in order to release the underlying io.ReadSeekCloser.
func (i *Index) Close() error {
	return i.f.Close()
}

// Open the named regular file for reading, returning an io.ReadCloser.
// Otherwise, return nil and an error. Please note that the caller is
// responsible to call [io.ReadCloser.Close] when done in order to release
// associated system resources.
func (i *Index) Open(name string) (io.ReadCloser, error) {
	el, ok := i.idx[name]
	if !ok {
		return nil, fmt.Errorf("tar file %q: no such element %q",
			i.f.Name(), name)
	}
	return NewPartialReader(i.f, el.Offset, el.Size)
}

// AllRegularFilePaths returns an iterator over the (unsorted) paths of all
// regular files in this Index.
func (i *Index) AllRegularFilePaths() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, el := range i.idx {
			if el.FileInfo().Mode() & ^fs.ModePerm != 0 {
				continue
			}
			if !yield(el.Name) {
				break
			}
		}
	}
}

// All returns an iterator over all paths and their FileInfo elements for
// regular files and directories in this tarball Index.
func (i *Index) All() iter.Seq2[string, fs.FileInfo] {
	return func(yield func(string, fs.FileInfo) bool) {
		for path := range maps.Keys(i.idx) {
			if !yield(path, i.idx[path].FileInfo()) {
				break
			}
		}
	}
}

// AllRegularFiles returns an iterator over all paths and their FileInfo
// elements of regular files in this tarball Index.
func (i *Index) AllRegularFiles() iter.Seq2[string, fs.FileInfo] {
	return func(yield func(string, fs.FileInfo) bool) {
		for path, info := range i.All() {
			if info.Mode() & ^fs.ModePerm != 0 {
				continue
			}
			if !yield(path, info) {
				break
			}
		}
	}
}
