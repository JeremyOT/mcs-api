/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2etest

import (
	"context"
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var (
	replicaCount = int32(1)
	helloService = v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "hello",
			},
			Ports: []v1.ServicePort{
				{
					Name:     "tcp",
					Port:     42,
					Protocol: v1.ProtocolTCP,
				},
				{
					Name:     "udp",
					Port:     42,
					Protocol: v1.ProtocolUDP,
				},
			},
		},
	}
	helloServiceImport = v1alpha1.ServiceImport{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello",
		},
		Spec: v1alpha1.ServiceImportSpec{
			Type: v1alpha1.ClusterSetIP,
			Ports: []v1alpha1.ServicePort{
				{
					Name:     "tcp",
					Port:     42,
					Protocol: v1.ProtocolTCP,
				},
				{
					Name:     "udp",
					Port:     42,
					Protocol: v1.ProtocolUDP,
				},
			},
		},
	}
	helloDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "hello",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "hello"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "hello-tcp",
							Image: "busybox",
							Args:  []string{"nc", "-lk", "-p", "42", "-v", "-e", "echo", "hello"},
						},
						{
							Name:  "hello-udp",
							Image: "busybox",
							Args:  []string{"nc", "-lk", "-p", "42", "-u", "-v", "-e", "echo", "hello"},
						},
					},
				},
			},
		},
	}
	requestPod = v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "request",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "request",
					Image: "busybox",
					Args:  []string{"nc"},
				},
			},
		},
	}
)

var _ = Describe("Connectivity", func() {
	var (
		namespace string

		ctx           = context.Background()
		serviceImport *v1alpha1.ServiceImport
	)
	BeforeEach(func() {
		namespace = fmt.Sprintf("e2etest-%v", rand.Uint32())
		_, err := cluster1.k8s.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = cluster2.k8s.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		dep := helloDeployment
		_, err = cluster2.k8s.AppsV1().Deployments(namespace).Create(ctx, &dep, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		svc := helloService
		_, err = cluster2.k8s.CoreV1().Services(namespace).Create(ctx, &svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		imp := helloServiceImport
		_, err = cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Create(ctx, &imp, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() int {
			slices, err := cluster2.k8s.DiscoveryV1beta1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labels.Set{discoveryv1beta1.LabelServiceName: helloService.Name}.AsSelector().String(),
			})
			Expect(err).ToNot(HaveOccurred())
			eps := 0
			for _, s := range slices.Items {
				eps += len(s.Endpoints)
			}
			return eps
		}, 30).Should(Equal(1))
		slices, err := cluster2.k8s.DiscoveryV1beta1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labels.Set{discoveryv1beta1.LabelServiceName: helloService.Name}.AsSelector().String(),
		})
		Expect(err).ToNot(HaveOccurred())
		importedSlice := slices.Items[0]
		importedSlice.ObjectMeta = metav1.ObjectMeta{
			Name: importedSlice.Name,
			Labels: map[string]string{
				v1alpha1.LabelServiceName: helloServiceImport.Name,
			},
		}
		_, err = cluster1.k8s.DiscoveryV1beta1().EndpointSlices(namespace).Create(ctx, &importedSlice, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() []string {
			svcImport, err := cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return svcImport.Spec.IPs
		}).ShouldNot(BeEmpty())
		serviceImport, err = cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() string {
			updatedSlice, err := cluster1.k8s.DiscoveryV1beta1().EndpointSlices(namespace).Get(ctx, importedSlice.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return updatedSlice.Labels[discoveryv1beta1.LabelServiceName]
		}).ShouldNot(BeEmpty())
		By("Created all in " + namespace)
	})
	AfterEach(func() {
		if *noTearDown {
			By(fmt.Sprintf("Skipping tearndown. Test namespace %q", namespace))
			By(fmt.Sprintf("Cluster 1: kubectl --kubeconfig %q -n %q", *kubeconfig1, namespace))
			By(fmt.Sprintf("Cluster 2: kubectl --kubeconfig %q -n %q", *kubeconfig2, namespace))
			return
		}
		Expect(cluster1.k8s.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}))
		Expect(cluster2.k8s.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}))
	})
	Specify("TCP connects across clusters using the VIP", func() {
		pod := requestPod
		pod.Spec.Containers[0].Args = []string{"nc", serviceImport.Spec.IPs[0], "42"}
		_, err := cluster1.k8s.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() (string, error) {
			return podLogs(ctx, cluster1.k8s, namespace, pod.Name)
		}, 30).Should(Equal("hello\n"))
	})
	Specify("UDP connects across clusters using the VIP", func() {
		pod := requestPod
		pod.Spec.Containers[0].Args = []string{"sh", "-c", fmt.Sprintf("echo hi | nc -u %s 42", serviceImport.Spec.IPs[0])}
		_, err := cluster1.k8s.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() (string, error) {
			return podLogs(ctx, cluster1.k8s, namespace, pod.Name)
		}, 30).Should(Equal("hello\n"))
	})
})
