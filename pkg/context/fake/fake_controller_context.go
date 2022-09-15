/*
Copyright 2019 The Kubernetes Authors.

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

import (
	//******What does this do?
	//Package record has all client logic for recording and
	//reporting "k8s.io/api/core/v1".Event events.
	clientrecord "k8s.io/client-go/tools/record"

	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/record"
)

// NewControllerContext returns a fake ControllerContext for unit testing
// reconcilers with a fake client.
func NewControllerContext(ctx *context.ControllerManagerContext) *context.ControllerContext {
	return &context.ControllerContext{
		ControllerManagerContext: ctx,
		Name:                     ControllerName,
		Logger:                   ctx.Logger.WithName(ControllerName),
		//How does this recorder is integrated and what does it record
		Recorder: record.New(clientrecord.NewFakeRecorder(1024)),
	}
}
