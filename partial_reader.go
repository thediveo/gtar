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
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// PartialReader is an io.ReaderCloser that returns only the contents of a range
// within an os.File.
type PartialReader struct {
	f         *os.File
	remaining int64
}

var _ io.ReadCloser = (*PartialReader)(nil)

// NewPartialReader returns an io.ReaderCloser-implementing object that reads
// and returns only the specified range within f. The PartialReader returned
// will use a duplicate of the passed file descriptor to a tar file. Thus, the
// caller must eventually call [PartialReader.Close] to release the underlying
// duplicated file descriptor when done.
func NewPartialReader(f *os.File, offset, length int64) (*PartialReader, error) {
	duplicatedfd, err := unix.Dup(int(f.Fd()))
	if err != nil {
		return nil, err
	}
	f = os.NewFile(uintptr(duplicatedfd),
		fmt.Sprintf("%s[%d:%d]", f.Name(), offset, offset+length))
	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}
	if stat, err := f.Stat(); err != nil || offset >= stat.Size() {
		f.Close()
		return nil, errors.New("cannot seek beyond EOF")
	}
	return &PartialReader{f: f, remaining: length}, nil
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0
// <= n <= len(p)) and any error encountered. It reads from the underlying file
// only within the range set when this PartialReader was created.
func (pr *PartialReader) Read(p []byte) (n int, err error) {
	if pr.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > pr.remaining {
		p = p[0:pr.remaining]
	}
	n, err = pr.f.Read(p)
	pr.remaining -= int64(n)
	return
}

// Close the PartialReader's underlying file descriptor in order to not leak
// system resources.
func (pr *PartialReader) Close() error {
	return pr.f.Close()
}
