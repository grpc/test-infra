package kubehelpers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("ContainerForName", func() {
	It("returns nil with an empty list", func() {
		actual := ContainerForName("match", []corev1.Container{})
		Expect(actual).To(BeNil())
	})

	It("returns nil without a matching name", func() {
		containers := []corev1.Container{
			{Name: "mismatch"},
		}

		actual := ContainerForName("match", containers)
		Expect(actual).To(BeNil())
	})

	It("returns a pointer to a container with a matching name", func() {
		containers := []corev1.Container{
			{Name: "match"},
			{Name: "mismatch"},
		}

		expected := &containers[0]
		actual := ContainerForName("match", containers)
		Expect(actual).To(BeIdenticalTo(expected))
	})
})
