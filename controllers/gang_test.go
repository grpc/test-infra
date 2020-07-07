package controllers

// import (
// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"

// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

// var _ = Describe("FindPools", func() {
// 	var (
// 		nodeList  = corev1.NodeList{}
// 		nodePools = []struct {
// 			pool   string
// 			length int
// 		}{
// 			{pool: "pool-a", length: 3},
// 			{pool: "pool-b", length: 1},
// 			{pool: "pool-c", length: 0},
// 		}
// 	)

// 	BeforeEach(func() {
// 		for _, nodePool := range nodePools {
// 			for l := 0; l < nodePool.length; l++ {
// 				nodeList.Items = append(nodeList.Items, corev1.Node{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Labels: map[string]string{
// 							"pool": nodePool.pool,
// 						},
// 					},
// 				})
// 			}
// 		}

// 		// one purposefully unlabeled node
// 		nodeList.Items = append(nodeList.Items, corev1.Node{})
// 	})

// 	It("counts nodes with identical pool labels", func() {
// 		pools, err := FindPools(nodeList)
// 		Expect(err).Should(BeNil())

// 		for _, nodePool := range nodePools {
// 			Expect(pools[nodePool.pool].Capacity).Should(Equal(nodePool.length))
// 		}
// 	})
// })
