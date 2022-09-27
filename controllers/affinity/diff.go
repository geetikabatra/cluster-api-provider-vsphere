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
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/services/govmomi/clustermodules"
)

type Diff struct {
	ToAdd    []Wrapper
	ToRemove []*ModuleInfo
	ToVerify []*ModuleInfo
}

type DiffResult struct {
	Items []diffResultItem
}

type diffResultItem struct {
	moduleInfo *infrav1.ClusterModule
	err        error
}

func (d DiffResult) Spec() []infrav1.ClusterModule {
	var mods []infrav1.ClusterModule
	for _, s := range d.Items {
		if s.moduleInfo != nil {
			mods = append(mods, *s.moduleInfo)
		}
	}
	return mods
}

func (d DiffResult) Error() error {
	var errList []error
	for _, s := range d.Items {
		if s.err != nil {
			errList = append(errList, s.err)
		}
	}
	return kerrors.NewAggregate(errList)
}

func (d Diff) Apply(ctx *context.AffinityContext) (*DiffResult, error) {
	result := &DiffResult{}
	// for each verify, verify and add
	d.verify(ctx, result)

	// for each add, create and add
	d.add(ctx, result)

	// for each remove, delete
	d.delete(ctx, result)

	return result, nil
}

// TODO (srm09): Add actual cluster module verify logic later
func (d Diff) verify(_ *context.AffinityContext, result *DiffResult) {
	for _, obj := range d.ToVerify {
		result.Items = append(result.Items, diffResultItem{
			moduleInfo: &infrav1.ClusterModule{
				ControlPlane:     obj.IsControlPlane(),
				TargetObjectName: obj.GetName(),
				ModuleUUID:       obj.ModuleUUID,
			},
		})
	}
}

func (d Diff) add(ctx *context.AffinityContext, result *DiffResult) {
	for _, obj := range d.ToAdd {
		vCenterSession, err := fetchSessionForObject(ctx, obj)
		if err != nil {
			result.Items = append(result.Items, diffResultItem{
				err: err,
			})
			continue
		}

		provider := clustermodules.NewProvider(vCenterSession.TagManager.Client)
		// TODO (srm09): How do we support Multi AZ scenarios here
		computeCluster, err := vCenterSession.Finder.ClusterComputeResourceOrDefault(ctx, "")
		if err != nil {
			result.Items = append(result.Items, diffResultItem{
				err: err,
			})
			continue
		}

		moduleUUID, err := provider.CreateModule(ctx, computeCluster.Reference())
		if err != nil {
			result.Items = append(result.Items, diffResultItem{
				err: err,
			})
			continue
		}
		//ctx.logger.Info("created cluster module for object", "moduleUUID", moduleUUID)
		result.Items = append(result.Items, diffResultItem{
			moduleInfo: &infrav1.ClusterModule{
				ControlPlane:     obj.IsControlPlane(),
				TargetObjectName: obj.GetName(),
				ModuleUUID:       moduleUUID,
			},
		})
	}
}

func (d Diff) delete(ctx *context.AffinityContext, result *DiffResult) error {
	params := newParams(*ctx)
	vcenterSession, err := fetchSession(ctx, params)
	if err != nil {
		return err
	}

	for _, moduleInfo := range d.ToRemove {
		provider := clustermodules.NewProvider(vcenterSession.TagManager.Client)
		logger := ctx.Logger.WithValues("moduleUUID", moduleInfo.ModuleUUID,
			"namespace", moduleInfo.GetNamespace(), "name", moduleInfo.GetName())
		if err := provider.DeleteModule(ctx, moduleInfo.ModuleUUID); err != nil {
			logger.Error(err, "failed to delete cluster module for object")
			result.Items = append(result.Items, diffResultItem{
				err: err,
			})
		} else {
			logger.Info("deleted cluster module for object")
		}
	}
	return nil
}

func diff(current CurrentState, desired DesiredState) Diff {
	var (
		toAdd    []Wrapper
		toRemove []*ModuleInfo
		toVerify []*ModuleInfo
	)

	if desired.ControlPlane == nil {
		if current.ControlPlane != nil {
			toRemove = append(toRemove, current.ControlPlane)
		}
	} else {
		if current.ControlPlane == nil {
			toAdd = append(toAdd, desired.ControlPlane)
		} else {
			toVerify = append(toVerify, current.ControlPlane)
		}
	}

	for _, obj := range desired.MachineDeployments {
		present, objWithMod := current.GetDeployment(obj.GetName())
		if !present {
			toAdd = append(toAdd, obj)
		} else {
			toVerify = append(toVerify, objWithMod)
		}
	}

	// these objects are no longer required, hence should be marked for deletion
	for _, objWithMod := range current.MachineDeployments {
		if !findInArr(toVerify, objWithMod.GetName()) {
			toRemove = append(toRemove, objWithMod)
		}
	}

	return Diff{
		ToAdd:    toAdd,
		ToRemove: toRemove,
		ToVerify: toVerify,
	}
}

func findInArr(arr []*ModuleInfo, key string) bool {
	found, _ := getInArr(arr, key)
	return found
}

func getInArr(arr []*ModuleInfo, key string) (bool, *ModuleInfo) {
	for _, objWithMod := range arr {
		if objWithMod.GetName() == key {
			return true, objWithMod
		}
	}
	return false, nil
}
