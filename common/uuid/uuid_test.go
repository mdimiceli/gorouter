package uuid_test

import (
	"github.com/mdimiceli/gorouter/common/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UUID", func() {
	It("creates a uuid", func() {
		uuid, err := uuid.GenerateUUID()
		Expect(err).ToNot(HaveOccurred())
		Expect(uuid).To(HaveLen(36))
	})
})
