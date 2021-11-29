/*


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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/prometheus/common/log"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	securityv1 "github.com/openshift/api/security/v1"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"

	"excelero.com/nvmesh-k8s-operator/controllers"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nvmeshv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func addToScheme(scheme *runtime.Scheme) {
	// This function adds custom objects to the scheme
	// this scheme will be used by the api client (watch / get /delete etc. ) and by the yaml decoder

	// Add CRDs type
	if err := apiext.AddToScheme(scheme); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	//Add OpenShift security to scheme
	// This is to allow us to handle OpenShift SecurityContextConstraints objects
	if err := securityv1.AddToScheme(scheme); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
}

func GetDynamicClientOrDie(config *rest.Config) dynamic.Interface {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "Unable to initialize dynamic client")
		os.Exit(1)
	}

	return client
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool

	operatorOptions := controllers.OperatorOptions{
		IsOpenShift: false,
		Development: false,
	}

	// metrics-addr default is 0 - so it is disable, if we want to re-enable it, we should set the default to :8080
	flag.StringVar(&metricsAddr, "metrics-addr", "0", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&operatorOptions.IsOpenShift, "openshift", false, "Set this flag if you are running on an openshift cluster")
	flag.StringVar(&operatorOptions.DefaultCoreImageTag, "core-image-tag", "tag-not-set", "The tag to use for the nvmesh core and utils images e.g. 0.7.0-4")

	// Development - Use this when developing locally and when you have access to the api-server but not internal ClusterIPs
	flag.BoolVar(&operatorOptions.Development, "development", false, "Used for development only")

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	addToScheme(scheme)

	setupLog.Info(fmt.Sprintf("operatorOptions.IsOpenShift: %t", operatorOptions.IsOpenShift))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "85de6a51.excelero.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	eventManager, err := controllers.NewEventManager(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "problem getting a client to update events on the CustomResource")
		os.Exit(1)
	}

	nvmeshReconciler := &controllers.NVMeshReconciler{
		NVMeshBaseReconciler: controllers.NVMeshBaseReconciler{
			Client:        mgr.GetClient(),
			Log:           ctrl.Log.WithName("controllers").WithName("NVMesh"),
			Scheme:        mgr.GetScheme(),
			DynamicClient: GetDynamicClientOrDie(mgr.GetConfig()),
			Manager:       mgr,
			EventManager:  eventManager,
			Options:       operatorOptions,
		},
	}

	if err = nvmeshReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NVMesh")
		os.Exit(1)
	}

	_, err = nvmeshReconciler.ListenToNodeLabels()
	if err != nil {
		setupLog.Error(err, "unable to create node listener", "controller", "NVMesh")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
