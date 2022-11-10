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

package e2e

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/mo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	capiutil "sigs.k8s.io/cluster-api/util"
)

var _ = FDescribe("Set Hardware Version Info", func() {
	var (
		namespace *corev1.Namespace
	)

	BeforeEach(func() {
		Expect(bootstrapClusterProxy).NotTo(BeNil(), "BootstrapClusterProxy can't be nil")
		namespace = setupSpecNamespace("hardware-version-e2e")
	})
	It("should create the cluster with worker nodes having Hardware Version set to vmx-17", func() {
		clusterName := fmt.Sprintf("cluster-%s", capiutil.RandomString(6))

		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   "hw",
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        e2eConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals("", "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals("", "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals("", "wait-worker-nodes"),
		}, &clusterctl.ApplyClusterTemplateAndWaitResult{})

		By("Verifying the Hardware Version of the worker node")
		VerifyHardwareVersion(clusterName, namespace.Name)
	})
	AfterEach(func() {
		cleanupSpecNamespace(namespace)
	})

})

func VerifyHardwareVersion(clusterName, namespace string) {
	list := getVSphereVMsForCluster(clusterName, namespace)
	for _, vm := range list.Items {
		if _, ok := vm.GetLabels()[v1beta1.MachineControlPlaneLabelName]; !ok {
			finder := find.NewFinder(vsphereClient.Client, false)
			dc, err := finder.Datacenter(ctx, vm.Spec.Datacenter)
			Expect(err).NotTo(HaveOccurred())
			finder.SetDatacenter(dc)

			vmObj, err := finder.VirtualMachine(ctx, fmt.Sprintf("/%s/vm/%s/%s", vm.Spec.Datacenter, vm.Spec.Folder, vm.Name))
			Expect(err).NotTo(HaveOccurred())
			var virtualMachine mo.VirtualMachine
			err = vmObj.Properties(ctx, vmObj.Reference(), []string{"config.version"}, &virtualMachine)
			Expect(err).NotTo(HaveOccurred())
			Expect(virtualMachine.Config.Version).To(Equal("vmx-17"))
		}
	}
}
