package extension_test

import (
	. "github.com/SUSE/eirinix"
	catalog "github.com/SUSE/eirinix/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Extension Manager", func() {
	c := catalog.NewCatalog()

	Context("Object creation", func() {
		manager := NewExtensionManager("namespace", "127.0.0.1", 90, nil)
		It("Is an interface", func() {
			m, ok := manager.(*DefaultExtensionManager)
			Expect(ok).To(Equal(true))
			Expect(m.Namespace).To(Equal("namespace"))
			Expect(m.Host).To(Equal("127.0.0.1"))
			Expect(m.Port).To(Equal(int32(90)))
		})

		It("Adds extensions", func() {
			manager.AddExtension(c.SimpleExtension())
			Expect(len(manager.ListExtensions())).To(Equal(1))
		})
	})
})
