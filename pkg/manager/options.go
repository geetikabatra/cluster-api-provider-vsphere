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
	"os"
	"strings"
	"time"

	//****** read about apimachinery run timex
	"k8s.io/apimachinery/pkg/runtime"

	// what is the diff between clinet-go/kubernetes and
	//client-go rest
	"k8s.io/client-go/rest"

	// ** see later where is this ues and update the comment here
	// This is used on 118 to extract the kubeconfig
	// a function Get configOrDie is used
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

// AddToManagerFunc is a function that can be optionally specified with
// the manager's Options in order to explicitly decide what controllers and
// webhooks to add to the manager.
type AddToManagerFunc func(*context.ControllerManagerContext, ctrlmgr.Manager) error

// Options describes the options used to create a new CAPV manager.
type Options struct {
	ctrlmgr.Options

	// EnableKeepAlive is a session feature to enable keep alive handler
	//*** what do we mean by enable keep alive handler here
	// for better load management on vSphere api server
	EnableKeepAlive bool

	// MaxConcurrentReconciles the maximum number of allowed, concurrent
	// reconciles.
	//
	// Defaults to the eponymous constant in this package.
	MaxConcurrentReconciles int

	// LeaderElectionNamespace is the namespace in which the pod running the
	// controller maintains a leader election lock
	//
	// Defaults to the eponymous constant in this package.
	PodNamespace string

	// PodName is the name of the pod running the controller manager.
	//
	// Defaults to the eponymous constant in this package.
	PodName string

	// Username is the username for the account used to access remote vSphere
	// endpoints.
	Username string

	// Password is the password for the account used to access remote vSphere
	// endpoints.
	Password string

	// KeepAliveDuration is the idle time interval in between send() requests
	// in keepalive handler
	// **********where is the send requiest coming from?
	//***** find more about keep alive handler
	KeepAliveDuration time.Duration

	// CredentialsFile is the file that contains credentials of CAPV
	CredentialsFile string

	KubeConfig *rest.Config

	// AddToManager is a function that can be optionally specified with
	// the manager's Options in order to explicitly decide what controllers
	// and webhooks to add to the manager.
	AddToManager AddToManagerFunc

	// NetworkProvider is the network provider used by Supervisor based clusters.
	// If not set, it will default to a DummyNetworkProvider which is intended for testing purposes.
	// VIM based clusters and managers will not need to set this flag.
	// ****** WHat is VIM based clusters
	// ******  Read more about Dummy network Provider
	NetworkProvider string
}

func (o *Options) defaults() {
	//***** what is a sink in logger
	// https://github.com/go-logr/logr says that logger simply fans out messages while
	//Sink lets you use the actual logging functionality, the difference is still
	//not clear
	if o.Logger.GetSink() == nil {
		o.Logger = ctrllog.Log
	}

	if o.PodName == "" {
		o.PodName = DefaultPodName
	}

	// GetConfigOrDie creates a *rest.Config for talking to a Kubernetes apiserver.
	// If --kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
	// in cluster and use the cluster provided kubeconfig.
	//
	// Will log an error and exit if there is an error creating the rest.Config
	// It will die if didn't get the config.
	// Still need to see how does it really get teh config
	if o.KubeConfig == nil {
		o.KubeConfig = config.GetConfigOrDie()
	}

	if o.Scheme == nil {
		o.Scheme = runtime.NewScheme()
	}

	if o.Username == "" || o.Password == "" {
		o.readAndSetCredentials()
	}

	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		o.PodNamespace = ns
		//see what is the significance of this file. Ask about this file
		// serach for this file here and read carefully
		// https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/
	} else if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			o.PodNamespace = ns
		}
	} else {
		o.PodNamespace = DefaultPodNamespace
	}
}

func (o *Options) getCredentials() map[string]string {
	file, err := os.ReadFile(o.CredentialsFile)
	if err != nil {
		o.Logger.Error(err, "error opening credentials file")
		return map[string]string{}
	}

	credentials := map[string]string{}
	if err := yaml.Unmarshal(file, &credentials); err != nil {
		o.Logger.Error(err, "error unmarshaling credentials to yaml")
		return map[string]string{}
	}

	return credentials
}

func (o *Options) readAndSetCredentials() {
	credentials := o.getCredentials()
	o.Username = credentials["username"]
	o.Password = credentials["password"]
}
