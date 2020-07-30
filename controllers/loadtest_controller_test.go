package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//corev1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Load Test Controller", func() {
	It("placeholder", func() {
		time.Sleep(5 * time.Second)
		err := k8sClient.Create(context.Background(), newLoadTest())
		Expect(err).ToNot(HaveOccurred())
	})
})
