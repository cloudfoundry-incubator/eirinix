package integration_test

import (
	//. "github.com/SUSE/eirinix"

	"time"

	catalog "github.com/SUSE/eirinix/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EiriniX", func() {
	Context("With a fake pod", func() {
		c := catalog.NewCatalog()
		It("Without an EiriniX extension running, it has only one environment variable", func() {
			app, err := c.StartEiriniApp()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				runs, err := app.IsRunning()
				Expect(err).ToNot(HaveOccurred())

				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

			err = app.Sync()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(app.Pod.Spec.Containers)).To(Equal(1))
			Expect(len(app.Pod.Spec.Containers[0].Envs)).To(Equal(1))
			Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "FAKE_APP", Value: "fake content"}))
			Expect(app.Delete()).ToNot(HaveOccurred())
		})
	})

	Context("With a simple extension running", func() {
		It("Starts, register the extension which injects a variable into the pod definition", func() {
			c := catalog.NewCatalog()
			m := c.IntegrationManager()
			e := &catalog.EditEnvExtension{}

			err := c.RegisterEiriniXService()
			Expect(err).ToNot(HaveOccurred())

			m.AddExtension(e)
			go m.Start()
			defer m.Stop() // Stop the extension when the test finishes
			//defer catalog.KubeClean()

			// At some point the extension should register
			Eventually(func() string {
				str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
				Expect(err).ToNot(HaveOccurred())

				return str
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).ShouldNot(ContainSubstring("No resources found in default namespace"))

			app, err := c.StartEiriniApp()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				runs, err := app.IsRunning()
				Expect(err).ToNot(HaveOccurred())

				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

			err = app.Sync()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(app.Pod.Spec.Containers)).To(Equal(1))
			Expect(len(app.Pod.Spec.Containers[0].Envs)).To(Equal(2))
			Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"}))
			Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "FAKE_APP", Value: "fake content"}))
			Expect(app.Delete()).ToNot(HaveOccurred())

			err = catalog.KubeClean()
			Expect(err).ToNot(HaveOccurred())

			str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
			Expect(err).ToNot(HaveOccurred())
			Expect(str).To(ContainSubstring("No resources found in default namespace"))
		})
	})

	Context("With a simple extension running", func() {
		It("Register the extension", func() {
			defer catalog.KubeClean() // Be sure to cleanup everything

			// Check nothing is left
			str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
			Expect(err).ToNot(HaveOccurred())
			Expect(str).To(ContainSubstring("No resources found in default namespace"))

			c := catalog.NewCatalog()
			m := c.IntegrationManager()
			e := &catalog.EditEnvExtension{}

			err = c.RegisterEiriniXService()
			Expect(err).ToNot(HaveOccurred())

			m.AddExtension(e)
			err = m.RegisterExtensions()
			Expect(err).ToNot(HaveOccurred())

			str, err = catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
			Expect(err).ToNot(HaveOccurred())
			Expect(str).ToNot(ContainSubstring("No resources found in default namespace"))

			m2 := c.IntegrationManagerNoRegister()
			m2.AddExtension(e)
			//	Expect(m2.Start()).ToNot(HaveOccurred())
			go m2.Start()   // we should first check registration is ok etc. But we need to setup services first
			defer m2.Stop() // Stop the extension when the test finishes
			time.Sleep(time.Second * 60)
			app, err := c.StartEiriniApp()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				runs, err := app.IsRunning()
				Expect(err).ToNot(HaveOccurred())

				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

			err = app.Sync()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(app.Pod.Spec.Containers)).To(Equal(1))
			Expect(len(app.Pod.Spec.Containers[0].Envs)).To(Equal(2))
			Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"}))
			Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "FAKE_APP", Value: "fake content"}))
			Expect(app.Delete()).ToNot(HaveOccurred())

		})
	})
})
