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

package govmomi_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/vmware/govmomi/simulator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/fake"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/services/govmomi"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/session"
	"sigs.k8s.io/cluster-api-provider-vsphere/test/helpers"
)

// @todo.
func TestReconcileVM(t *testing.T) {
	g := NewWithT(t)
	model := simulator.VPX()
	model.Host = 0 // ClusterHost only
	simr, err := helpers.VCSimBuilder().WithModel(model).Build()
	// g.Expect(err).ToNot(HaveOccurred())
	if err != nil {
		t.Fatalf("unable to create simulator: %s", err)
	}
	defer simr.Destroy()
	// sim, err := helpers.VCSimBuilder().Build()
	// g.Expect(err).ToNot(HaveOccurred())

	// t.Cleanup(func() {
	// 	sim.Destroy()
	// })

	vms := govmomi.VMService{}
	// @todo: refactor subtests to desc-fn-expect []struct

	// Case 1: error if vm is "in flight", it must return an error --
	// ? how do we configure a preflight task on the fake context?
	t.Run("when vm context has an inflight task", func(t *testing.T) {
		// g := NewWithT(t)
		vmContext := fake.NewVMContext(fake.NewControllerContext(fake.NewControllerManagerContext()))
		vmContext.VSphereVM.Spec.Server = simr.ServerURL().Host
		authSession, err := session.GetOrCreate(
			vmContext.Context,
			session.NewParams().
				WithServer(vmContext.VSphereVM.Spec.Server).
				WithUserInfo(simr.Username(), simr.Password()).
				WithDatacenter("*"))
		if err != nil {
			t.Fatal(err)
		}
		vmContext.Session = authSession
		vm := simulator.Map.Any("VirtualMachine").(*simulator.VirtualMachine)
		vmContext.VSphereVM.Spec.Template = vm.Name
		// vmCtx, err := getFakeContext(sim)
		// g.Expect(err).ToNot(HaveOccurred())
		vmContext.VSphereVM.Status = infrav1.VSphereVMStatus{
			TaskRef:    "some-inflight-task",
			RetryAfter: metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
		}

		vmtest, err := vms.ReconcileVM(vmContext)

		g.Expect(err).To(HaveOccurred())
		g.Expect(vmtest).To(Equal(infrav1.VirtualMachine{
			Name:  vmContext.VSphereVM.Name,
			State: infrav1.VirtualMachineStatePending,
		}))
	})

	// // Case 2: Returns error on failure to find VM by BiosUUID (pass an invalid UUID?)
	// t.Run("when vm reference is set but it cannot be found", func(t *testing.T) {

	// })

	// // Case 3: Bootstraps new VM when the VM doesn't exist already
	// t.Run("when vm does not exist already", func(t *testing.T) {

	// })

	// ...
}

// @todo.
// func TestDestroyVM(t *testing.T) {

// }

// // func getFakeContext(sim *helpers.Simulator) (*context.VMContext, error) {
// // 	ctx := fake.NewVMContext(fake.NewControllerContext(fake.NewControllerManagerContext()))
// // 	ctx.VSphereVM.Spec.Server = sim.ServerURL().Host

// // 	authSession, err := session.GetOrCreate(
// // 		ctx,
// // 		session.NewParams().
// // 			WithServer(ctx.VSphereVM.Spec.Server).
// // 			WithUserInfo(sim.Username(), sim.Password()).
// // 			WithDatacenter("*"))
// // 	if err != nil {
// // 		return nil, err
// // 	}

// // 	ctx.Session = authSession

// // 	return ctx, nil
// // }

// // nolint:deadcode,unused
// // @todo: remove nolint decl.
// func getFakeTask(state types.TaskInfoState, errorDescription string) mo.Task {
// 	t := mo.Task{
// 		ExtensibleManagedObject: mo.ExtensibleManagedObject{
// 			Self: types.ManagedObjectReference{
// 				Value: "-for-logger",
// 			},
// 		},
// 	}
// 	if state != "" {
// 		t.Info = types.TaskInfo{
// 			State: state,
// 		}
// 	}
// 	if errorDescription != "" {
// 		t.Info.Description = &types.LocalizableMessage{
// 			Message: errorDescription,
// 		}
// 	}
// 	return t
// }
