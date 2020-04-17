module github.com/SUSE/eirinix

require (
	code.cloudfoundry.org/cf-operator v0.4.0
	github.com/appscode/jsonpatch v0.0.0-20190108182946-7c0e3b262f30
	github.com/cloudflare/cfssl v0.0.0-20181102015659-ea4033a214e7
	github.com/google/martian v2.1.0+incompatible
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.9.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.8.1
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	go.uber.org/zap v1.10.0
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a // indirect
	google.golang.org/appengine v1.5.0 // indirect
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.2.0
	sigs.k8s.io/yaml v1.1.0
)

go 1.13
