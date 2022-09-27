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

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

func TestCurrentState_GetDeployment(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		uuidMap map[string]string
		found   bool
	}{
		{
			name:    "with no machine deployment",
			uuidMap: map[string]string{},
			key:     "foo",
		},
		{
			name: "with single machine deployment & found",
			key:  "foo-one",
			uuidMap: map[string]string{
				"foo-one": "uuid-one",
			},
			found: true,
		},
		{
			name: "with single machine deployment & not found",
			key:  "bar",
			uuidMap: map[string]string{
				"foo-one": "uuid-one",
			},
		},
		{
			name: "with multiple machine deployments & found",
			key:  "foo-two",
			uuidMap: map[string]string{
				"foo-one":   "uuid-one",
				"foo-two":   "uuid-two",
				"foo-three": "uuid-three",
			},
			found: true,
		},
		{
			name: "with multiple machine deployments & not found",
			key:  "bar",
			uuidMap: map[string]string{
				"foo-one":   "uuid-one",
				"foo-two":   "uuid-two",
				"foo-three": "uuid-three",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			var mods []*ModuleInfo
			for name, uid := range tt.uuidMap {
				mods = append(mods, &ModuleInfo{
					Wrapper: mdWrapper{&clusterv1.MachineDeployment{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: metav1.NamespaceDefault,
							Name:      name,
						},
					}},
					ModuleUUID: uid,
				})
			}
			currState := CurrentState{
				MachineDeployments: mods,
			}

			found, val := currState.GetDeployment(tt.key)
			g.Expect(tt.found).To(Equal(found))
			if tt.found {
				g.Expect(val.ModuleUUID).To(Equal(tt.uuidMap[tt.key]))
			}
		})
	}
}

func TestReconciler_reconcileCurrentState(t *testing.T) {
	type ctrlPlaneInput struct {
		name string
		uid  string
	}
	tests := []struct {
		name         string
		uuidMap      map[string]string
		controlPlane *ctrlPlaneInput
	}{
		{
			name:         "single control plane & machine deployment",
			controlPlane: &ctrlPlaneInput{util.RandomString(10), uuid.New().String()},
			uuidMap: map[string]string{
				util.RandomString(10): uuid.New().String(),
			},
		},
		{
			name:         "single control plane & multiple machine deployments",
			controlPlane: &ctrlPlaneInput{util.RandomString(10), uuid.New().String()},
			uuidMap: map[string]string{
				util.RandomString(10): uuid.New().String(),
				util.RandomString(10): uuid.New().String(),
				util.RandomString(10): uuid.New().String(),
			},
		},
		{
			name:         "single control plane & no machine deployment",
			controlPlane: &ctrlPlaneInput{util.RandomString(10), uuid.New().String()},
		},
		{
			name: "no control plane & single machine deployment",
			uuidMap: map[string]string{
				util.RandomString(10): uuid.New().String(),
			},
		},
		{
			name: "no control plane & multiple machine deployments",
			uuidMap: map[string]string{
				util.RandomString(10): uuid.New().String(),
				util.RandomString(10): uuid.New().String(),
				util.RandomString(10): uuid.New().String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			var mods []infrav1.ClusterModule
			if tt.controlPlane != nil {
				mods = append(mods, clusterModule(tt.controlPlane.name, tt.controlPlane.uid, true))
			}
			for k, v := range tt.uuidMap {
				mods = append(mods, clusterModule(k, v, false))
			}
			vsphereCluster := &infrav1.VSphereCluster{
				ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceDefault},
				Spec: infrav1.VSphereClusterSpec{
					ClusterModules: mods,
				},
			}

			currState := Reconciler{}.reconcileCurrentState(&context.AffinityContext{
				VSphereCluster: vsphereCluster,
			})

			if tt.controlPlane != nil {
				g.Expect(currState.ControlPlane.GetName()).To(Equal(tt.controlPlane.name))
				g.Expect(currState.ControlPlane.ModuleUUID).To(Equal(tt.controlPlane.uid))
				g.Expect(currState.ControlPlane.GetNamespace()).To(Equal(metav1.NamespaceDefault))
			}
			g.Expect(currState.MachineDeployments).To(HaveLen(len(tt.uuidMap)))
			for _, actualMD := range currState.MachineDeployments {
				g.Expect(tt.uuidMap).To(HaveKeyWithValue(actualMD.GetName(), actualMD.ModuleUUID))
				g.Expect(actualMD.GetNamespace()).To(Equal(metav1.NamespaceDefault))
			}
		})
	}
}

func clusterModule(name, uid string, controlPlane bool) infrav1.ClusterModule {
	return infrav1.ClusterModule{
		ControlPlane:     controlPlane,
		ModuleUUID:       uid,
		TargetObjectName: name,
	}
}
