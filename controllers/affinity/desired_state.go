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
	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

type DesiredState struct {
	ControlPlane Wrapper

	MachineDeployments []Wrapper
}

func (r Reconciler) reconcileDesiredState(ctx *context.AffinityContext) (*DesiredState, error) {
	name, ok := ctx.VSphereCluster.GetLabels()[clusterv1.ClusterLabelName]
	if !ok {
		return nil, errors.Errorf("missing CAPI cluster label")
	}

	labels := map[string]string{clusterv1.ClusterLabelName: name}
	kcpList := &controlplanev1.KubeadmControlPlaneList{}
	if err := r.Client.List(
		ctx, kcpList,
		client.InNamespace(ctx.VSphereCluster.GetNamespace()),
		client.MatchingLabels(labels)); err != nil {
		return nil, errors.Wrapf(err, "failed to list control plane objects")
	}
	if len(kcpList.Items) > 1 {
		return nil, errors.Errorf("multiple control plane objects found, expected 1, found %d", len(kcpList.Items))
	}

	desiredState := &DesiredState{}
	if len(kcpList.Items) != 0 {
		if kcp := &kcpList.Items[0]; kcp.GetDeletionTimestamp().IsZero() {
			desiredState.ControlPlane = kcpWrapper{kcp}
		}
	}

	mdList := &clusterv1.MachineDeploymentList{}
	if err := r.Client.List(
		ctx, mdList,
		client.InNamespace(ctx.VSphereCluster.GetNamespace()),
		client.MatchingLabels(labels)); err != nil {
		return nil, errors.Wrapf(err, "failed to list machine deployment objects")
	}
	for _, md := range mdList.Items {
		if md.DeletionTimestamp.IsZero() {
			desiredState.MachineDeployments = append(desiredState.MachineDeployments, mdWrapper{md.DeepCopy()})
		}
	}

	return desiredState, nil
}
