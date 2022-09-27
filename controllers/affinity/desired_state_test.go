/*
Copyright 2022 The Kubernetes Authors.

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

package affinity

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/fake"
)

func TestReconciler_getDesiredState(t *testing.T) {
	tests := []struct {
		name         string
		numOfMDs     int
		hasError     bool
		initObjs     []client.Object
		customAssert func(g *WithT, state *DesiredState)
	}{
		{
			name: "multiple control planes",
			initObjs: []client.Object{
				controlPlane("foo-1", metav1.NamespaceDefault, fake.Clusterv1a2Name),
				controlPlane("foo-2", metav1.NamespaceDefault, fake.Clusterv1a2Name),
			},
			hasError: true,
		},
		{
			name:     "single control plane & no machine deployment",
			initObjs: []client.Object{controlPlane("foo", metav1.NamespaceDefault, fake.Clusterv1a2Name)},
			numOfMDs: 0,
		},
		{
			name: "single control plane & machine deployment",
			initObjs: []client.Object{
				controlPlane("foo", metav1.NamespaceDefault, fake.Clusterv1a2Name),
				machineDeployment("foo", metav1.NamespaceDefault, fake.Clusterv1a2Name),
				machineDeployment("foo", "bar", fake.Clusterv1a2Name),
			},
			numOfMDs: 1,
		},
		{
			name: "single control plane & multiple machine deployments",
			initObjs: []client.Object{
				controlPlane("foo", metav1.NamespaceDefault, fake.Clusterv1a2Name),
				machineDeployment("foo-1", metav1.NamespaceDefault, fake.Clusterv1a2Name),
				machineDeployment("foo-2", metav1.NamespaceDefault, fake.Clusterv1a2Name),
				machineDeployment("foo", "bar", fake.Clusterv1a2Name),
			},
			numOfMDs: 2,
			customAssert: func(g *WithT, state *DesiredState) {
				var names []string
				for _, md := range state.MachineDeployments {
					names = append(names, md.GetName())
				}
				g.Expect(names).To(ConsistOf("foo-1", "foo-2"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controllerCtx := fake.NewControllerContext(fake.NewControllerManagerContext(tt.initObjs...))
			ctx := fake.NewAffinityContext(controllerCtx)

			r := Reconciler{controllerCtx}
			desiredState, err := r.reconcileDesiredState(ctx)
			if tt.hasError {
				g.Expect(err).To(HaveOccurred())
				return
			}

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(desiredState.MachineDeployments).To(HaveLen(tt.numOfMDs))
			if tt.customAssert != nil {
				tt.customAssert(g, desiredState)
			}
		})
	}

	t.Run("with objects marked for deletion", func(t *testing.T) {
		g := NewWithT(t)

		currTime := metav1.Now()
		mdToBeDeleted := machineDeployment("foo", metav1.NamespaceDefault, fake.Clusterv1a2Name)
		mdToBeDeleted.DeletionTimestamp = &currTime

		controllerCtx := fake.NewControllerContext(fake.NewControllerManagerContext(
			controlPlane("foo", metav1.NamespaceDefault, fake.Clusterv1a2Name),
			machineDeployment("foo-1", metav1.NamespaceDefault, fake.Clusterv1a2Name),
			mdToBeDeleted,
		))
		ctx := fake.NewAffinityContext(controllerCtx)

		desiredState, err := Reconciler{controllerCtx}.reconcileDesiredState(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(desiredState.MachineDeployments).To(HaveLen(1))
	})
}

func machineDeployment(name, namespace, cluster string) *clusterv1.MachineDeployment {
	return &clusterv1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{clusterv1.ClusterLabelName: cluster},
		},
	}
}

func controlPlane(name, namespace, cluster string) *controlplanev1.KubeadmControlPlane {
	return &controlplanev1.KubeadmControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{clusterv1.ClusterLabelName: cluster},
		},
	}
}
