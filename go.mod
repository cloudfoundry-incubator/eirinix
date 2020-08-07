module github.com/SUSE/eirinix

require (
	code.cloudfoundry.org/cf-operator v1.0.1-0.20200413083459-fb39a29ad746
	code.cloudfoundry.org/quarks-secret v1.0.699
	code.cloudfoundry.org/quarks-utils v0.0.0-20200721113854-0b1ab7e84ec5
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.2.2
	go.uber.org/zap v1.15.0
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
)

replace code.cloudfoundry.org/cf-operator => code.cloudfoundry.org/quarks-operator v1.0.1-0.20200413083459-fb39a29ad746

go 1.13
