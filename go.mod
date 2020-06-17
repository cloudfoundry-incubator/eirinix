module github.com/SUSE/eirinix

replace code.cloudfoundry.org/cf-operator v1.0.1-0.20200413083459-fb39a29ad746 => code.cloudfoundry.org/quarks-operator v1.0.1-0.20200413083459-fb39a29ad746

require (
	code.cloudfoundry.org/cf-operator v1.0.1-0.20200413083459-fb39a29ad746
	code.cloudfoundry.org/quarks-utils v0.0.0-20200331122601-bc0838ffea60
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.2.2
	go.uber.org/zap v1.15.0
	k8s.io/api v0.0.0-20200404061942-2a93acf49b83
	k8s.io/apimachinery v0.0.0-20200410010401-7378bafd8ae2
	k8s.io/client-go v0.0.0-20200330143601-07e69aceacd6
	sigs.k8s.io/controller-runtime v0.4.0
)

go 1.13
