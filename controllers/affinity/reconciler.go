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
	goctx "context"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

type Reconciler struct {
	*context.ControllerContext
}

func (r Reconciler) Reconcile(ctx goctx.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := r.Logger.WithName(req.Namespace).WithName(req.Name)
	logger.Info("affinity reconcile", "request", req.NamespacedName)

	vsphereCluster := &infrav1.VSphereCluster{}
	if err := r.Client.Get(r, client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, vsphereCluster); err != nil {
		r.Logger.Error(err, "failed to get vSphereCluster object")
		return reconcile.Result{}, err
	}

	patchHelper, err := patch.NewHelper(vsphereCluster, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(
			err,
			"failed to init patch helper for %s %s/%s",
			vsphereCluster.GroupVersionKind(),
			vsphereCluster.Namespace,
			vsphereCluster.Name)
	}

	affinityCtx := &context.AffinityContext{
		ControllerContext: r.ControllerContext,
		VSphereCluster:    vsphereCluster,
		PatchHelper:       patchHelper,
		Logger:            logger,
	}
	defer func() {
		if err := affinityCtx.Patch(); err != nil {
			if reterr == nil {
				reterr = err
			}
			r.Logger.Error(err, "patch failed", "vspherecluster", vsphereCluster.Name)
		}
	}()

	return r.reconcileNormal(affinityCtx)
}

func (r Reconciler) reconcileNormal(ctx *context.AffinityContext) (reconcile.Result, error) {
	logger := ctx.Logger.WithValues("namespace", ctx.VSphereCluster.Namespace, "cluster", ctx.VSphereCluster.Name)
	logger.Info("reconcile normal")

	ctrlutil.AddFinalizer(ctx.VSphereCluster, infrav1.ClusterAffinityFinalizer)

	currentState := r.reconcileCurrentState(ctx)

	desiredState, err := r.reconcileDesiredState(ctx)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	}

	result, err := r.calculateDiff(ctx, currentState, *desiredState)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	}

	if err := result.Error(); err != nil {
		logger.Error(err, "errors when reconciling the cluster modules")
	}
	ctx.VSphereCluster.Spec.ClusterModules = result.Spec()
	if len(result.Spec()) == 0 {
		ctrlutil.RemoveFinalizer(ctx.VSphereCluster, infrav1.ClusterAffinityFinalizer)
	}
	return reconcile.Result{}, nil
}

func (r Reconciler) ToAffinityInput(obj client.Object) []reconcile.Request {
	cluster, err := util.GetClusterFromMetadata(r, r.Client, metav1.ObjectMeta{
		Namespace:       obj.GetNamespace(),
		Labels:          obj.GetLabels(),
		OwnerReferences: obj.GetOwnerReferences(),
	})
	if err != nil {
		r.Logger.Error(err, "failed to get owner cluster")
		return nil
	}

	vsphereCluster := &infrav1.VSphereCluster{}
	if err := r.Client.Get(r, client.ObjectKey{
		Name:      cluster.Spec.InfrastructureRef.Name,
		Namespace: cluster.Namespace,
	}, vsphereCluster); err != nil {
		r.Logger.Error(err, "failed to get vSphereCluster object",
			"namespace", cluster.Namespace, "name", cluster.Spec.InfrastructureRef.Name)
		return nil
	}

	return []reconcile.Request{
		{client.ObjectKeyFromObject(vsphereCluster)},
	}
}
