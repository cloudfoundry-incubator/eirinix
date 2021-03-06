package integration_test

import (
	"time"

	extension "code.cloudfoundry.org/eirinix"
	catalog "code.cloudfoundry.org/eirinix/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var _ = Describe("EiriniX", func() {

	var (
		cat catalog.Catalog
		mgr extension.Manager
		ext *catalog.EditEnvExtension
	)

	BeforeEach(func() {
		cat = catalog.NewCatalog()
		mgr = cat.IntegrationManager()
		ext = &catalog.EditEnvExtension{}
	})

	AfterEach(func() {
		mgr.Stop()
		Expect(catalog.KubeClean()).To(Succeed())
		ExtensionShouldBeUnregistered()
	})

	Context("extensions", func() {
		var app *catalog.EiriniApp

		JustBeforeEach(func() {
			var err error
			app, err = cat.StartEiriniApp()
			Expect(err).ToNot(HaveOccurred())
			Eventually(app.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())
		})

		When("there is no EiriniX extension running", func() {
			It("does not change the app", func() {
				AppShouldHaveSingleContainerWithEnv(app, []catalog.ContainerEnv{
					{Name: "FAKE_APP", Value: "fake content"},
				})
			})
		})

		When("there is a simple extension running in the default namespace", func() {
			BeforeEach(func() {
				Expect(cat.RegisterEiriniXService()).To(Succeed())

				mgr.AddExtension(ext)
				go mgr.Start()

				EventuallyExtensionShouldBeRegistered()
			})

			It("adds the env var", func() {
				AppShouldHaveSingleContainerWithEnv(app, []catalog.ContainerEnv{
					{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"},
					{Name: "FAKE_APP", Value: "fake content"},
				})
			})

			When("the app is created in a non-default namespace", func() {
				var (
					othernamespace string
					otherApp       *catalog.EiriniApp
				)

				BeforeEach(func() {
					othernamespace = "othernamespace"
					createNamespace(othernamespace)

					var err error
					otherApp, err = cat.StartEiriniAppInNamespace(othernamespace)
					Expect(err).ToNot(HaveOccurred())
					Eventually(otherApp.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())
				})

				AfterEach(func() {
					deleteNamespace(othernamespace)
				})

				It("does not add the env var", func() {
					AppShouldHaveSingleContainerWithEnv(otherApp, []catalog.ContainerEnv{
						{Name: "FAKE_APP", Value: "fake content"},
					})
				})

			})
		})

		When("the extension is separately registered and started", func() {
			var m2 extension.Manager

			BeforeEach(func() {
				ExtensionShouldBeUnregistered()

				Expect(cat.RegisterEiriniXService()).To(Succeed())

				mgr.AddExtension(ext)
				Expect(mgr.RegisterExtensions()).To(Succeed())

				EventuallyExtensionShouldBeRegistered()

				m2 = cat.IntegrationManagerNoRegister()
				m2.AddExtension(ext)

				go m2.Start()

				EventuallyExtensionShouldBe("eirini-x-mutating-hook")
			})

			AfterEach(func() {
				m2.Stop()
			})

			It("adds the env var", func() {
				AppShouldHaveSingleContainerWithEnv(app, []catalog.ContainerEnv{
					{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"},
					{Name: "FAKE_APP", Value: "fake content"},
				})
			})
		})

		When("the extension listens on all namespaces", func() {

			var (
				othernamespace string
				otherApp       *catalog.EiriniApp
			)

			BeforeEach(func() {
				Expect(cat.RegisterEiriniXService()).To(Succeed())

				mgr = cat.IntegrationManagerFiltered(false, "")
				mgr.AddExtension(ext)

				go mgr.Start()

				EventuallyExtensionShouldBeRegistered()

				othernamespace = "othernamespace"
				createNamespace(othernamespace)

				var err error
				otherApp, err = cat.StartEiriniAppInNamespace(othernamespace)

				Expect(err).ToNot(HaveOccurred())
				Eventually(otherApp.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())
			})

			AfterEach(func() {
				deleteNamespace(othernamespace)
			})

			It("adds the env var to the othernamespace app", func() {
				AppShouldHaveSingleContainerWithEnv(otherApp, []catalog.ContainerEnv{
					{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"},
					{Name: "FAKE_APP", Value: "fake content"},
				})
			})

			It("still adds the env var to the original app", func() {
				AppShouldHaveSingleContainerWithEnv(app, []catalog.ContainerEnv{
					{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"},
					{Name: "FAKE_APP", Value: "fake content"},
				})
			})

		})
	})

	Context("watchers", func() {
		var (
			resultChan chan watch.Event
			watcher    extension.Watcher
		)

		BeforeEach(func() {
			ExtensionShouldBeUnregistered()

			resultChan = make(chan watch.Event, 3)
			watcher = cat.SimpleWatcherWithChannel(resultChan)

		})

		JustBeforeEach(func() {
			mgr.AddWatcher(watcher)
			go mgr.Watch()
		})

		When("watching for Eirini apps only", func() {

			BeforeEach(func() {
				mgr = cat.IntegrationManagerFiltered(true, "default")
			})

			It("is notified about the pod", func() {
				app, err := cat.StartEiriniApp()
				Expect(err).ToNot(HaveOccurred())
				Eventually(app.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

				Expect(app.Delete()).To(Succeed())

				EventuallyReceivesEvent(watcher, watch.Added, app.Name)
				EventuallyReceivesEvent(watcher, watch.Modified, app.Name)
				EventuallyReceivesEvent(watcher, watch.Modified, app.Name)
			})
		})

		When("the watcher doesn't filter pods", func() {
			var (
				testns string
			)

			BeforeEach(func() {
				testns = "watchertest"
				createNamespace(testns)

				mgr = cat.IntegrationManagerFiltered(false, testns)
			})

			AfterEach(func() {
				deleteNamespace(testns)
			})

			It("can see all pods in the namespace", func() {
				app, err := cat.StartEiriniStagingAppInNamespace(testns)
				Expect(err).ToNot(HaveOccurred())
				Eventually(app.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

				Expect(app.Delete()).To(Succeed())

				EventuallyReceivesEvent(watcher, watch.Added, app.Name)
				EventuallyReceivesEvent(watcher, watch.Modified, app.Name)
				EventuallyReceivesEvent(watcher, watch.Modified, app.Name)
			})
		})

		When("the watcher filters Eirini app pods", func() {

			var (
				listenNamespace string
				otherNamespace  string
			)

			BeforeEach(func() {
				ExtensionShouldBeUnregistered()

				listenNamespace = "listen-here"
				createNamespace(listenNamespace)

				otherNamespace = "go-away"
				createNamespace(otherNamespace)

				mgr = cat.IntegrationManagerFiltered(true, listenNamespace)
			})

			AfterEach(func() {
				deleteNamespace(listenNamespace)
				deleteNamespace(otherNamespace)
			})

			It("can see only Eirini pods in the namespace", func() {
				staging, err := cat.StartEiriniStagingAppInNamespace(listenNamespace)
				Expect(err).ToNot(HaveOccurred())
				Eventually(staging.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

				standardapp, err := cat.StartEiriniAppInNamespace(listenNamespace)
				Expect(err).ToNot(HaveOccurred())
				Eventually(standardapp.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

				EventuallyReceivesEvent(watcher, watch.Added, standardapp.Name)
				EventuallyReceivesEvent(watcher, watch.Modified, standardapp.Name)
				EventuallyReceivesEvent(watcher, watch.Modified, standardapp.Name)
			})

			It("cannot see Eirini pods in other namespaces when namespace is set", func() {
				otherApp, err := cat.StartEiriniAppInNamespace(otherNamespace)
				Expect(err).ToNot(HaveOccurred())
				Eventually(otherApp.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

				extension, _ := watcher.(*catalog.SimpleWatcherWithChannel)
				Consistently(extension.Received, "5s").ShouldNot(Receive())
			})

			When("the namespace is not set", func() {
				BeforeEach(func() {
					mgr = cat.IntegrationManagerFiltered(true, "")
				})

				It("can see pods in other namespaces", func() {
					app, err := cat.StartEiriniAppInNamespace(otherNamespace)
					Expect(err).ToNot(HaveOccurred())
					Eventually(app.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())

					EventuallyReceivesEvent(watcher, watch.Added, app.Name)
					EventuallyReceivesEvent(watcher, watch.Modified, app.Name)
					EventuallyReceivesEvent(watcher, watch.Modified, app.Name)
				})
			})
		})

		When("watching in a single namespace", func() {
			var testns string

			BeforeEach(func() {
				testns = "watchertest"
				createNamespace(testns)

				mgr = cat.IntegrationManagerFiltered(false, testns)
				go mgr.Start()
			})

			AfterEach(func() {
				deleteNamespace(testns)
			})

			It("maps the operator to that namespace via a label in the watched namespace", func() {
				EventuallyWithOffset(1, func() (string, error) {
					return catalog.Kubectl([]string{}, "get", "namespace", testns, "-o", "jsonpath='{.metadata.labels.eirini-x-ns}'")
				}, "1m").Should(ContainSubstring(testns))
			})
		})
	})

	Context("reconcilers", func() {
		var app *catalog.EiriniApp

		JustBeforeEach(func() {
			var err error
			app, err = cat.StartEiriniApp()
			Expect(err).ToNot(HaveOccurred())
			Eventually(app.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())
		})

		When("there is a simple extension running in the default namespace", func() {
			BeforeEach(func() {
				Expect(cat.RegisterEiriniXService()).To(Succeed())

				mgr.AddReconciler(cat.SimpleReconciler())
				go mgr.Start()

				EventuallyExtensionShouldBeRegistered()
			})

			It("adds the annotation", func() {
				Expect(app.Sync()).To(Succeed())
				Expect(app.Pod.Metadata.Annotations).To(ContainElement("yes"))
				Expect(app.Pod.Metadata.Annotations["touched"]).To(Equal("yes"))
			})
		})
	})

	Context("multiple extension types", func() {
		var (
			app *catalog.EiriniApp

			resultChan chan watch.Event
			watcher    extension.Watcher
		)

		JustBeforeEach(func() {
			var err error
			app, err = cat.StartEiriniApp()
			Expect(err).ToNot(HaveOccurred())
			Eventually(app.IsRunning, time.Duration(60*time.Second), time.Duration(5*time.Second)).Should(BeTrue())
		})

		When("there are extension, watcher and reconcilers started from the same process", func() {
			BeforeEach(func() {
				resultChan = make(chan watch.Event, 3)
				watcher = cat.SimpleWatcherWithChannel(resultChan)

				Expect(cat.RegisterEiriniXService()).To(Succeed())

				mgr.AddExtension(&catalog.EditImageReconciler{})
				mgr.AddExtension(watcher)
				mgr.AddExtension(ext)

				go mgr.Start()

				EventuallyExtensionShouldBeRegistered()
			})

			It("adds the annotation", func() {
				Expect(app.Sync()).To(Succeed())
				Eventually(func() string {
					app.Sync()
					return app.Pod.Spec.Containers[0].Image
				}, "60s").Should(Equal("opensuse/leap"))
			})

			It("still adds the env var to the original app", func() {
				AppShouldHaveSingleContainerWithEnv(app, []catalog.ContainerEnv{
					{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"},
					{Name: "FAKE_APP", Value: "fake content"},
				})
			})

			It("processes events", func() {
				EventuallyReceivesEvent(watcher, watch.Added, app.Name)
				EventuallyReceivesEvent(watcher, watch.Modified, app.Name)
				EventuallyReceivesEvent(watcher, watch.Modified, app.Name)
			})
		})
	})
})

func createNamespace(name string) {
	_, err := catalog.Kubectl([]string{}, "create", "namespace", name)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func deleteNamespace(name string) {
	_, err := catalog.Kubectl([]string{}, "delete", "namespace", name)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func EventuallyExtensionShouldBeRegistered() {
	EventuallyWithOffset(1, func() (string, error) {
		return catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
	}, time.Duration(60*time.Second), time.Duration(5*time.Second)).ShouldNot(ContainSubstring("No resources found"))
}

func EventuallyExtensionShouldBe(name string) {
	EventuallyWithOffset(1, func() (string, error) {
		return catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
	}, "60s").Should(ContainSubstring(name))
}

func ExtensionShouldBeUnregistered() {
	str, err := catalog.Kubectl([]string{}, "get", "mutatingwebhookconfiguration")
	ExpectWithOffset(1, str).To(ContainSubstring("No resources found"))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func AppShouldHaveSingleContainerWithEnv(app *catalog.EiriniApp, contents []catalog.ContainerEnv) {
	ExpectWithOffset(1, app.Sync()).To(Succeed())
	ExpectWithOffset(1, app.Pod.Spec.Containers).To(HaveLen(1))
	ExpectWithOffset(1, app.Pod.Spec.Containers[0].Envs).To(ConsistOf(contents))
}

func EventuallyReceivesEvent(watcher extension.Watcher, evType watch.EventType, appName string) {
	extension, _ := watcher.(*catalog.SimpleWatcherWithChannel)

	ev := &watch.Event{}
	EventuallyWithOffset(1, extension.Received, "5s").Should(Receive(ev))
	ExpectWithOffset(1, ev.Type).To(Equal(evType))

	pod, ok := ev.Object.(*corev1.Pod)
	ExpectWithOffset(1, ok).To(BeTrue())
	ExpectWithOffset(1, pod.GetName()).To(Equal(appName))
}
