// Package testing contains methods to create test data. It's a seaparate
// package to avoid import cycles. Helper functions can be found in the package
// `testhelper`.
package testing

import (
	"context"

	operator_testing "code.cloudfoundry.org/cf-operator/testing"
	eirinix "github.com/SUSE/eirinix"
)

// NewCatalog returns a Catalog, our helper for test cases
func NewCatalog() Catalog {
	return Catalog{Catalog: &operator_testing.Catalog{}}
}

// NewContext returns a non-nil empty context, for usage when it is unclear
// which context to use.  Mostly used in tests.
func NewContext() context.Context {
	return operator_testing.NewContext()
}

// Catalog provides several instances for test, based on the cf-operator's catalog
type Catalog struct{ *operator_testing.Catalog }

// SimpleExtension it's returning a fake dummy Eirini extension
func (c *Catalog) SimpleExtension() eirinix.Extension {

	return &testExtension{
		parentExtension{Name: "test"}}
}

// SimpleManager returns a dummy Extensions manager
func (c *Catalog) SimpleManager() eirinix.Manager {
	return eirinix.NewManager(
		eirinix.ManagerOptions{
			Namespace: "namespace",
			Host:      "127.0.0.1",
			Port:      90,
		})
}
