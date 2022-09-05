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

package manager

import (
	goctx "context"
	"fmt"
	"os"

	"github.com/pkg/errors"

	//see what does this do?

	netopv1 "github.com/vmware-tanzu/net-operator-api/api/v1alpha1"
	vmoprv1 "github.com/vmware-tanzu/vm-operator-api/api/v1alpha1"

	//what is ncpv
	ncpv1 "github.com/vmware-tanzu/vm-operator/external/ncp/api/v1alpha1"
	topologyv1 "github.com/vmware-tanzu/vm-operator/external/tanzu-topology/api/v1alpha1"
	"gopkg.in/fsnotify.v1"

	//read about clinet go scheme
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"

	//write small definition of controller runtime here
	ctrl "sigs.k8s.io/controller-runtime"

	infrav1a3 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1alpha3"
	infrav1a4 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1alpha4"
	infrav1b1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	vmwarev1b1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/record"
)

// Manager is a CAPV controller manager.
type Manager interface {
	ctrl.Manager

	// GetContext returns the controller manager's context.
	//*********walk through the context path
	GetContext() *context.ControllerManagerContext
}

// New returns a new CAPV controller manager.
func New(opts Options) (Manager, error) {
	// Ensure the default options are set.
	//***********What do we mean by default options?
	opts.defaults()
	//****** _ is a blank iidnetifier which means we donot want to use the rsult

	//******** what exactly do we mean by scheme
	_ = clientgoscheme.AddToScheme(opts.Scheme)
	_ = clusterv1.AddToScheme(opts.Scheme)
	_ = infrav1a3.AddToScheme(opts.Scheme)
	_ = infrav1a4.AddToScheme(opts.Scheme)
	_ = infrav1b1.AddToScheme(opts.Scheme)
	_ = bootstrapv1.AddToScheme(opts.Scheme)
	_ = vmwarev1b1.AddToScheme(opts.Scheme)
	_ = vmoprv1.AddToScheme(opts.Scheme)
	_ = ncpv1.AddToScheme(opts.Scheme)
	_ = netopv1.AddToScheme(opts.Scheme)
	_ = topologyv1.AddToScheme(opts.Scheme)
	//**************Read about what scaffold does
	// +kubebuilder:scaffold:scheme

	// ******* Hostname could be anything, why this specifically.
	podName, err := os.Hostname()
	if err != nil {
		podName = DefaultPodName
	}

	// Build the controller manager.
	mgr, err := ctrl.NewManager(opts.KubeConfig, opts.Options)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create manager")
	}

	// Build the controller manager context.
	controllerManagerContext := &context.ControllerManagerContext{
		Context: goctx.Background(),
		//what is the difference between watchnamespace and namespace

		WatchNamespace: opts.Namespace,
		Namespace:      opts.PodNamespace,
		Name:           opts.PodName,

		//****** read about why leader election ID is required by controller
		//runtime
		LeaderElectionID:        opts.LeaderElectionID,
		LeaderElectionNamespace: opts.LeaderElectionNamespace,
		//****** See how would this affect the performance
		MaxConcurrentReconciles: opts.MaxConcurrentReconciles,
		Client:                  mgr.GetClient(),
		Logger:                  opts.Logger.WithName(opts.PodName),
		//*** what is the significance of recorder and what
		//particular events is this recording
		//see how does this work
		Recorder: record.New(mgr.GetEventRecorderFor(fmt.Sprintf("%s/%s", opts.PodNamespace, podName))),
		//**** what scheme really does?
		//Scheme defines methods to serialize and desirialize API objects,
		// it basically converts gvk(group version kind ) to and from go schema
		// Go schema is basically used to convert the fomm values into struc
		//values.(read properly)
		//https://github.com/kubernetes/apimachinery/blob/master/pkg/runtime/scheme.go#L31-L34
		//https://pkg.go.dev/github.com/gorilla/schema#section-readme
		// type is a struct and
		Scheme:          opts.Scheme,
		Username:        opts.Username,
		Password:        opts.Password,
		EnableKeepAlive: opts.EnableKeepAlive,
		//***** If enable keepAlive is false, what is the default value of
		//Keepalive duration
		KeepAliveDuration: opts.KeepAliveDuration,

		//********What does network provider has a role here
		NetworkProvider: opts.NetworkProvider,
	}

	// Add the requested items to the manager.
	if err := opts.AddToManager(controllerManagerContext, mgr); err != nil {
		return nil, errors.Wrap(err, "failed to add resources to the manager")
	}

	// *********** what does scaffold: builder does
	// +kubebuilder:scaffold:builder

	return &manager{
		Manager: mgr,
		ctx:     controllerManagerContext,
	}, nil
}

type manager struct {
	// k8s manager is used as a composition here
	ctrl.Manager
	ctx *context.ControllerManagerContext
}

func (m *manager) GetContext() *context.ControllerManagerContext {
	return m.ctx
}

func UpdateCredentials(opts *Options) {
	opts.readAndSetCredentials()
}

// InitializeWatch adds a filesystem watcher for the capv credentials file
// In case of any update to the credentials file, the new credentials are passed to the capv manager context.
// ***** this is a good method. you can use it in many places
func InitializeWatch(ctx *context.ControllerManagerContext, managerOpts *Options) (watch *fsnotify.Watcher, err error) {
	capvCredentialsFile := managerOpts.CredentialsFile
	updateEventCh := make(chan bool)
	watch, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to create new Watcher for %s", capvCredentialsFile))
	}
	if err = watch.Add(capvCredentialsFile); err != nil {
		return nil, errors.Wrap(err, "received error on CAPV credential watcher")
	}
	go func() {
		for {
			select {
			//**********read about channels again in go
			case err := <-watch.Errors:
				ctx.Logger.Error(err, "received error on CAPV credential watcher")
			case event := <-watch.Events:
				ctx.Logger.Info(fmt.Sprintf("received event %v on the credential file %s", event, capvCredentialsFile))
				updateEventCh <- true
			}
		}
	}()

	go func() {
		for range updateEventCh {
			// ******* this didnot make sense, when are
			//the new credentials initialized
			UpdateCredentials(managerOpts)
		}
	}()

	return watch, err
}
