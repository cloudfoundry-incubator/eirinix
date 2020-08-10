package integration_test

import (
	"time"

	extension "github.com/SUSE/eirinix"
	catalog "github.com/SUSE/eirinix/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var _ = Describe("EiriniX", func() {

	var (
		c catalog.Catalog
		m extension.Manager
		e *catalog.EditEnvExtension
	)

	BeforeEach(func() {
		catalog.KubeClean() // Be sure to cleanup everything
		c = catalog.NewCatalog()
		m = c.IntegrationManager()
		e = &catalog.EditEnvExtension{}

	})

	Context("injecting a variable into the pod definition", func() {
		var app *catalog.EiriniApp

		JustBeforeEach(func() {
			var err error
			app, err = c.StartEiriniApp()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				runs, err := app.IsRunning()
				Expect(err).ToNot(HaveOccurred())

				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())
		})

		AfterEach(func() {
			Expect(catalog.KubeClean()).To(Succeed())

			str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
			Expect(err).ToNot(HaveOccurred())
			Expect(str).To(ContainSubstring("No resources found in default namespace"))
		})

		When("there is no EiriniX extension running", func() {
			It("has only one environment variable", func() {
				Expect(app.Sync()).To(Succeed())
				Expect(app.Pod.Spec.Containers).To(HaveLen(1))
				Expect(app.Pod.Spec.Containers[0].Envs).To(HaveLen(1))
				Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "FAKE_APP", Value: "fake content"}))
			})
		})

		When("there is a simple extension running", func() {
			BeforeEach(func() {
				err := c.RegisterEiriniXService()
				Expect(err).ToNot(HaveOccurred())

				m.AddExtension(e)
				go m.Start()

				// At some point the extension should register
				Eventually(func() string {
					str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
					Expect(err).ToNot(HaveOccurred())

					return str
				}, time.Duration(60*time.Second), time.Duration(5*time.Second)).ShouldNot(ContainSubstring("No resources found in default namespace"))
			})

			AfterEach(func() {
				m.Stop()
			})

			It("adds the env var", func() {
				Expect(app.Sync()).To(Succeed())
				Expect(app.Pod.Spec.Containers).To(HaveLen(1))
				Expect(app.Pod.Spec.Containers[0].Envs).To(HaveLen(2))
				Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"}))
				Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "FAKE_APP", Value: "fake content"}))
			})
		})

		When("the extension is separately registered and started", func() {
			var m2 extension.Manager

			BeforeEach(func() {
				str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
				Expect(err).ToNot(HaveOccurred())
				Expect(str).To(ContainSubstring("No resources found in default namespace"))

				err = c.RegisterEiriniXService()
				Expect(err).ToNot(HaveOccurred())

				m.AddExtension(e)
				err = m.RegisterExtensions()
				Expect(err).ToNot(HaveOccurred())

				str, err = catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
				Expect(err).ToNot(HaveOccurred())
				Expect(str).ToNot(ContainSubstring("No resources found in default namespace"))

				m2 = c.IntegrationManagerNoRegister()
				m2.AddExtension(e)
				//	Expect(m2.Start()).ToNot(HaveOccurred())
				go m2.Start() // we should first check registration is ok etc. But we need to setup services first

				Eventually(func() (string, error) {
					return catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
				}, "60s").Should(ContainSubstring("eirini-x-mutating-hook"))
			})

			AfterEach(func() {
				m2.Stop()
			})

			It("adds the env var", func() {
				Expect(app.Sync()).To(Succeed())
				Expect(app.Pod.Spec.Containers).To(HaveLen(1))
				Expect(app.Pod.Spec.Containers[0].Envs).To(HaveLen(2))
				Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"}))
				Expect(app.Pod.Spec.Containers[0].Envs).Should(ContainElement(catalog.ContainerEnv{Name: "FAKE_APP", Value: "fake content"}))
			})

		})
	})

	Context("With a watcher for Eirini Apps only", func() {
		BeforeEach(func() {
			m = c.IntegrationManagerFiltered(true, "default")
		})

		It("can see pods in the namespace", func() {
			// Check nothing is left
			str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
			Expect(err).ToNot(HaveOccurred())
			Expect(str).To(ContainSubstring("No resources found in default namespace"))

			resultChan := make(chan watch.Event, 3) // Test will check first 3 events
			w := c.SimpleWatcherWithChannel(resultChan)

			m.AddWatcher(w)
			go m.Watch() // Start the watchers

			// we shouldn't have any webhook registered
			str, err = catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
			Expect(err).ToNot(HaveOccurred())
			Expect(str).To(ContainSubstring("No resources found in default namespace"))

			// Generate 3 events (ADD,MODIFIED,MODIFIED)
			app, err := c.StartEiriniApp()
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				runs, err := app.IsRunning()
				Expect(err).ToNot(HaveOccurred())
				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

			Expect(app.Delete()).ToNot(HaveOccurred()) // Delete the app (multiple events MODIFIED should be triggered)

			time.Sleep(time.Second * 5) // Give time to the watcher to process the events

			extension, _ := w.(*catalog.SimpleWatcherWithChannel)
			// Consume the recorded event
			for _, ev := range []string{"ADDED", "MODIFIED", "MODIFIED"} {
				event, ok := <-extension.Received

				Expect(string(event.Type)).To(Equal(ev))
				Expect(ok).To(BeTrue())

				pod, ok := event.Object.(*corev1.Pod)
				Expect(ok).To(BeTrue())
				Expect(pod.GetName()).To(Equal(app.Name))
			}
			m.Stop()
		})
	})

	Context("With a watcher that doesn't filter pods", func() {
		testns := "watchertest"

		BeforeEach(func() {
			m = c.IntegrationManagerFiltered(false, testns)
		})

		It("can see all pods in the namespace", func() {

			defer func() {
				_, err := catalog.Kubectl([]string{}, "delete", "namespace", testns)
				Expect(err).ToNot(HaveOccurred())
			}()

			_, err := catalog.Kubectl([]string{}, "create", "namespace", testns)
			Expect(err).ToNot(HaveOccurred())

			resultChan := make(chan watch.Event, 3) // Test has 3 events
			w := c.SimpleWatcherWithChannel(resultChan)

			m.AddWatcher(w)

			go m.Watch() // Start the watchers

			// Generate 3 events (ADD,MODIFIED,MODIFIED)
			app, err := c.StartEiriniStagingAppInNamespace(testns)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				runs, err := app.IsRunning()
				Expect(err).ToNot(HaveOccurred())
				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

			Expect(app.Delete()).ToNot(HaveOccurred()) // Delete the app (multiple events MODIFIED should be triggered)

			time.Sleep(time.Second * 5) // Give time to the watcher to process the events

			extension, _ := w.(*catalog.SimpleWatcherWithChannel)
			// Consume the recorded event
			for _, ev := range []string{"ADDED", "MODIFIED", "MODIFIED"} {
				event, ok := <-extension.Received
				Expect(ok).To(BeTrue())

				pod, ok := event.Object.(*corev1.Pod)
				Expect(ok).To(BeTrue())
				Expect(pod.GetName()).To(Equal(app.Name))

				Expect(string(event.Type)).To(Equal(ev))
				Expect(ok).To(BeTrue())
			}

			m.Stop()
		})
	})

	Context("With a watcher that filters Eirini app pods", func() {
		It("can see only Eirini pods in the namespace", func() {
			testns := "watchertest2"
			m = c.IntegrationManagerFiltered(true, testns)

			defer func() {
				_, err := catalog.Kubectl([]string{}, "delete", "namespace", testns)
				Expect(err).ToNot(HaveOccurred())
			}()

			_, err := catalog.Kubectl([]string{}, "create", "namespace", testns)
			Expect(err).ToNot(HaveOccurred())
			// Check nothing is left
			str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
			Expect(err).ToNot(HaveOccurred())
			Expect(str).To(ContainSubstring("No resources found in default namespace"))

			resultChan := make(chan watch.Event, 10) // Test has 3 events

			w := c.SimpleWatcherWithChannel(resultChan)

			m.AddWatcher(w)

			go m.Watch() // Start the watchers

			staging, err := c.StartEiriniStagingAppInNamespace(testns)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				runs, err := staging.IsRunning()
				Expect(err).ToNot(HaveOccurred())
				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

			standardapp, err := c.StartEiriniAppInNamespace(testns)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				runs, err := standardapp.IsRunning()
				Expect(err).ToNot(HaveOccurred())
				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

			time.Sleep(time.Second * 5) // Give time to the watcher to process the events

			extension, _ := w.(*catalog.SimpleWatcherWithChannel)
			// Consume the recorded event - and there should be only one

			// Consume the recorded event
			for _, ev := range []string{"ADDED", "MODIFIED", "MODIFIED"} {
				event, ok := <-extension.Received
				Expect(ok).To(BeTrue())

				pod, ok := event.Object.(*corev1.Pod)
				Expect(ok).To(BeTrue())
				Expect(pod.GetName()).To(Equal(standardapp.Name))

				Expect(string(event.Type)).To(Equal(ev))
				Expect(ok).To(BeTrue())
			}

			m.Stop()
		})

		It("cannot see Eirini pods in other namespaces when namespace is set", func() {
			appns := "app-namespace"
			m = c.IntegrationManagerFiltered(true, "listen-namespace")

			defer func() {
				_, err := catalog.Kubectl([]string{}, "delete", "namespace", appns)
				Expect(err).ToNot(HaveOccurred())
			}()

			_, err := catalog.Kubectl([]string{}, "create", "namespace", appns)
			Expect(err).ToNot(HaveOccurred())
			// Check nothing is left
			str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
			Expect(err).ToNot(HaveOccurred())
			Expect(str).To(ContainSubstring("No resources found in default namespace"))

			resultChan := make(chan watch.Event, 10) // Test has 3 events

			w := c.SimpleWatcherWithChannel(resultChan)

			m.AddWatcher(w)

			go m.Watch() // Start the watchers

			standardapp, err := c.StartEiriniAppInNamespace(appns)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				runs, err := standardapp.IsRunning()
				Expect(err).ToNot(HaveOccurred())
				return runs
			}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

			extension, _ := w.(*catalog.SimpleWatcherWithChannel)
			Consistently(extension.Received, "5s").ShouldNot(Receive())
			m.Stop()
		})

		Context("with namespace unset", func() {
			It("can see pods in other namespaces", func() {
				testns := "nonlabeledtestns"
				m = c.IntegrationManagerFiltered(true, "") // Monitor all namespaces

				defer func() {
					_, err := catalog.Kubectl([]string{}, "delete", "namespace", testns)
					Expect(err).ToNot(HaveOccurred())
				}()

				_, err := catalog.Kubectl([]string{}, "create", "namespace", testns)
				Expect(err).ToNot(HaveOccurred())
				// Check nothing is left
				str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
				Expect(err).ToNot(HaveOccurred())
				Expect(str).To(ContainSubstring("No resources found in default namespace"))

				resultChan := make(chan watch.Event, 10) // Test has 3 events

				w := c.SimpleWatcherWithChannel(resultChan)

				m.AddWatcher(w)

				go m.Watch() // Start the watchers

				staging, err := c.StartEiriniStagingAppInNamespace(testns)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					runs, err := staging.IsRunning()
					Expect(err).ToNot(HaveOccurred())
					return runs
				}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

				standardapp, err := c.StartEiriniAppInNamespace(testns)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					runs, err := standardapp.IsRunning()
					Expect(err).ToNot(HaveOccurred())
					return runs
				}, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

				time.Sleep(time.Second * 5) // Give time to the watcher to process the events

				extension, _ := w.(*catalog.SimpleWatcherWithChannel)
				// Consume the recorded event - and there should be only one

				// Consume the recorded event
				for _, ev := range []string{"ADDED", "MODIFIED", "MODIFIED"} {
					event, ok := <-extension.Received
					Expect(ok).To(BeTrue())

					pod, ok := event.Object.(*corev1.Pod)
					Expect(ok).To(BeTrue())
					Expect(pod.GetName()).To(Equal(standardapp.Name))

					Expect(string(event.Type)).To(Equal(ev))
					Expect(ok).To(BeTrue())
				}

				m.Stop()
			})
		})
	})
})
