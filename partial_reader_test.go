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
	"io"
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("reading only parts of files", Ordered, func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Filedescriptors).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
				ShouldNot(HaveLeakedFds(goodfds))
		})
	})

	Context("error handling", func() {

		It("handles closed files", func() {
			closedf := Successful(os.Open("/dev/null"))
			closedf.Close()
			Expect(NewPartialReader(closedf, 0, 42)).Error().To(HaveOccurred())
		})

		It("cannot start outside the file", func() {
			f := Successful(os.Open("./testdata/somefile"))
			fclose := sync.OnceFunc(func() {
				Expect(f.Close()).To(Succeed())
			})
			defer fclose()
			Expect(NewPartialReader(f, 666, 1)).Error().To(HaveOccurred())
			Expect(NewPartialReader(f, -1, 1)).Error().To(HaveOccurred())
		})

	})

	It("reads only part of a file", func() {
		f := Successful(os.Open("./testdata/somefile"))
		defer f.Close()
		r := Successful(NewPartialReader(f, 8, 4))
		defer r.Close()
		Expect(io.ReadAll(r)).To(Equal([]byte("89ab")))
	})

})
