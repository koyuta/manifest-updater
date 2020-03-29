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
	"context"
	"flag"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	manifestupdaterkoyutaiov1alpha1 "manifest-updater/api/v1alpha1"
	"manifest-updater/controllers"
	"manifest-updater/updater"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = manifestupdaterkoyutaiov1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr string
		interval    uint
		token       string
	)
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.UintVar(&interval, "interval", 60, "")
	flag.StringVar(&token, "token", "", "")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     true,
		LeaderElectionID:   "manifest-updater-leader",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	queue := make(chan *updater.Entry, 1)

	if err = (&controllers.UpdaterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Updater"),
		Scheme: mgr.GetScheme(),
		Queue:  queue,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Updater")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	looper := updater.NewUpdateLooper(
		queue,
		time.Duration(interval)*time.Second,
		ctrl.Log.WithName("Loop"),
		token,
	)

	var (
		loopStop = make(chan struct{}, 1)
		mgrStop  = make(chan struct{}, 1)
	)
	go func() {
		<-ctrl.SetupSignalHandler()
		loopStop <- struct{}{}
		mgrStop <- struct{}{}
	}()

	var eg, _ = errgroup.WithContext(context.Background())
	setupLog.Info("starting looper")
	eg.Go(func() error {
		return looper.Loop(loopStop)
	})
	setupLog.Info("starting manager")
	eg.Go(func() error {
		return mgr.Start(mgrStop)
	})
	if err := eg.Wait(); err != nil {
		setupLog.Error(err, "problem running manager")
	}
}
