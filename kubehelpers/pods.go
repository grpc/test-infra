package kubehelpers

import (
	corev1 "k8s.io/api/core/v1"
)

// ContainerForName accepts a string and a slice of containers. It returns a
// pointer to the container with a name that matches the string. If no names
// match, it returns nil.
func ContainerForName(name string, containers []corev1.Container) *corev1.Container {
	for i := range containers {
		container := &containers[i]

		if container.Name == name {
			return container
		}
	}

	return nil
}
