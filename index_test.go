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
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

func Keys[K, V any](it iter.Seq2[K, V]) iter.Seq[K] {
	return func(yield func(K) bool) {
		for key := range it {
			if !yield(key) {
				break
			}
		}
	}
}

var _ = Describe("reading only parts of files", Ordered, func() {

	var tarballPath string

	BeforeAll(func() {
		By("creating a tarball for testing")
		tf := Successful(os.CreateTemp("", "testing-tar-*"))
		defer tf.Close() // sic!
		tarballPath = tf.Name()
		DeferCleanup(func() {
			By("removing testing tarball")
			Expect(os.Remove(tarballPath)).To(Succeed())
		})
		tarw := tar.NewWriter(tf)
		defer tarw.Close()
		Expect(filepath.Walk("./testdata/tar", func(path string, info fs.FileInfo, err error) error {
			hdr := Successful(tar.FileInfoHeader(info, ""))
			hdr.Name = filepath.Clean(path)
			Expect(tarw.WriteHeader(hdr)).To(Succeed())
			if info.IsDir() {
				return nil
			}
			f := Successful(os.Open(path))
			defer f.Close()
			Expect(io.Copy(tarw, f)).Error().NotTo(HaveOccurred())
			return nil
		})).To(Succeed())
	})

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Filedescriptors).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
				ShouldNot(HaveLeakedFds(goodfds))
		})
	})

	Context("errors", func() {

		It("cannot create an index from a non-existing file", func() {
			Expect(New("./testdata/nada-nothing-nix")).Error().To(HaveOccurred())
		})

		It("cannot create an index using a close file", func() {
			f := Successful(os.Open("/dev/null"))
			f.Close()
			Expect(NewFromFile(f)).Error().To(HaveOccurred())
		})

		It("cannot open non-existing files", func() {
			i := Successful(New(tarballPath))
			defer i.Close()

			Expect(i.Open("rumpelpumpel")).Error().To(HaveOccurred())
		})

	})

	Context("aborting iterators", func() {

		It("All", func() {
			i := Successful(New(tarballPath))
			defer i.Close()
			count := 0
			for range i.All() {
				count++
				break
			}
			Expect(count).To(Equal(1))
		})

		It("AllRegularFiles", func() {
			i := Successful(New(tarballPath))
			defer i.Close()
			count := 0
			for range i.AllRegularFiles() {
				count++
				break
			}
			Expect(count).To(Equal(1))
		})

		It("AllRegularFilePaths", func() {
			i := Successful(New(tarballPath))
			defer i.Close()
			count := 0
			for range i.AllRegularFilePaths() {
				count++
				break
			}
			Expect(count).To(Equal(1))
		})

	})

	It("creates an index and reads files", func() {
		i := Successful(New(tarballPath))
		defer i.Close()

		Expect(i.AllRegularFilePaths()).To(ConsistOf(
			"testdata/tar/foo",
			"testdata/tar/bar/baz",
		))

		Expect(Keys(i.All())).To(ConsistOf(
			"testdata/tar",
			"testdata/tar/foo",
			"testdata/tar/bar",
			"testdata/tar/bar/baz",
		))

		Expect(Keys(i.AllRegularFiles())).To(ConsistOf(
			"testdata/tar/foo",
			"testdata/tar/bar/baz",
		))

		f := Successful(i.Open("testdata/tar/foo"))
		defer f.Close()
		Expect(io.ReadAll(f)).To(Equal([]byte("foo")))

		f = Successful(i.Open("testdata/tar/bar/baz"))
		defer f.Close()
		Expect(io.ReadAll(f)).To(Equal([]byte("1234567890")))
	})

})
