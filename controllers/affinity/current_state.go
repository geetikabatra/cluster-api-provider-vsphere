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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

type ModuleInfo struct {
	Wrapper

	ModuleUUID string
}

type CurrentState struct {
	ControlPlane *ModuleInfo

	MachineDeployments []*ModuleInfo
}

func (c CurrentState) GetDeployment(name string) (bool, *ModuleInfo) {
	return getInArr(c.MachineDeployments, name)
}

// TODO (srm09): Add actual object fetching logic later, will need to consider case where object is deleted.
func (r Reconciler) reconcileCurrentState(ctx *context.AffinityContext) CurrentState {
	currentState := CurrentState{}

	for _, mod := range ctx.VSphereCluster.Spec.ClusterModules {
		objMetadata := metav1.ObjectMeta{
			Namespace: ctx.VSphereCluster.Namespace,
			Name:      mod.TargetObjectName,
		}
		if mod.ControlPlane {
			currentState.ControlPlane = &ModuleInfo{
				Wrapper: kcpWrapper{&controlplanev1.KubeadmControlPlane{
					ObjectMeta: objMetadata,
				}},
				ModuleUUID: mod.ModuleUUID,
			}
		} else {
			currentState.MachineDeployments = append(currentState.MachineDeployments, &ModuleInfo{
				Wrapper: mdWrapper{&clusterv1.MachineDeployment{
					ObjectMeta: objMetadata,
				}},
				ModuleUUID: mod.ModuleUUID,
			})
		}
	}
	return currentState
}
