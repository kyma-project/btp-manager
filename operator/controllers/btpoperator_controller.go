/*
Copyright 2022.

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

package controllers

import (
	"fmt"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	"github.com/kyma-project/module-manager/operator/pkg/declarative"
	"github.com/kyma-project/module-manager/operator/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorName = "btp-manager"
	labelKey     = "app.kubernetes.io/managed-by"
	chartPath    = "./module-chart"
)

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	declarative.ManifestReconciler
	*rest.Config
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;get;list;watch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *BtpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()
	if err := r.initReconciler(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BtpOperator{}).
		Complete(r)
}

func (r *BtpOperatorReconciler) initReconciler(mgr ctrl.Manager) error {
	manifestResolver := &ManifestResolver{}
	return r.Inject(mgr, &v1alpha1.BtpOperator{},
		declarative.WithManifestResolver(manifestResolver),
		declarative.WithCustomResourceLabels(map[string]string{labelKey: operatorName}),
		// declarative.WithPostRenderTransform(transform),
		// declarative.WithResourcesReady(true),
	)
}

/*
// transform modifies the resources based on some criteria, before installation.
func transform(_ context.Context, _ types.BaseCustomObject, manifestResources *types.ManifestResources) error {

}
*/

// ManifestResolver represents the chart information for the passed Sample resource.
type ManifestResolver struct{}

// Get returns the chart information to be processed.
func (m *ManifestResolver) Get(obj types.BaseCustomObject) (types.InstallationSpec, error) {
	btpServiceOperator, valid := obj.(*v1alpha1.BtpOperator)
	if !valid {
		return types.InstallationSpec{},
			fmt.Errorf("invalid type conversion for %s", client.ObjectKeyFromObject(obj))
	}
	return types.InstallationSpec{
		ChartPath:   chartPath,
		ReleaseName: btpServiceOperator.Spec.ReleaseName,
	}, nil
}
