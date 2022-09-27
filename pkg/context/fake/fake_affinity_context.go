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

package fake

import "sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"

func NewAffinityContext(ctx *context.ControllerContext) *context.AffinityContext {
	vsphereCluster := newVSphereCluster(newClusterV1())
	if err := ctx.Client.Create(ctx, &vsphereCluster); err != nil {
		panic(err)
	}

	return &context.AffinityContext{
		ControllerContext: ctx,
		VSphereCluster:    &vsphereCluster,
		Logger:            ctx.Logger.WithName(vsphereCluster.Name),
	}
}
