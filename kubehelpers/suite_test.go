package kubehelpers

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestKubehelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubehelpers Suite")
}
