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
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	"github.com/kyma-project/module-manager/operator/pkg/custom"
	"github.com/kyma-project/module-manager/operator/pkg/manifest"
	"github.com/kyma-project/module-manager/operator/pkg/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	chartPath         = "./module-chart"
	chartNamespace    = "kyma-system"
	operatorName      = "btp-manager"
	labelKeyForChart  = "app.kubernetes.io/managed-by"
	secretNameField   = ".metadata.name"
	secretLabelKey    = "kyma-project.io/provided-by"
	secretLabelValue  = "kyma-environment-broker"
	secretName        = "btp-manager-secret"
	deletionFinalizer = "custom-deletion-finalizer"
	requeueInterval   = time.Second * 5
)

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	*rest.Config
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BtpOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here
	btpOperatorCr := &v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, btpOperatorCr); err != nil {
		logger.Error(err, "unable to fetch BtpOperator")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if _, err := r.getRequiredSecret(ctx); err != nil {
		logger.Error(err, "while getting required Secret")
		return ctrl.Result{}, err
	}

	if ctrlutil.AddFinalizer(btpOperatorCr, deletionFinalizer) {
		return ctrl.Result{}, r.Update(ctx, btpOperatorCr)
	}

	switch btpOperatorCr.Status.State {
	case "":
		return ctrl.Result{}, r.HandleInitialState(ctx, btpOperatorCr)
	case types.StateProcessing:
		return ctrl.Result{RequeueAfter: requeueInterval}, r.HandleProcessingState(ctx, btpOperatorCr)
	}

	/*
		var existingBtpOperators v1alpha1.BtpOperatorList
		if err := r.List(ctx, &existingBtpOperators); err != nil {
			logger.Error(err, "unable to fetch existing BtpOperators")
			return ctrl.Result{}, err
		}
	*/

	return ctrl.Result{}, nil
}

func (r *BtpOperatorReconciler) getRequiredSecret(ctx context.Context) (v1.Secret, error) {
	secrets := &v1.SecretList{}
	if err := r.List(ctx, secrets, client.MatchingFields{secretNameField: secretName}, client.MatchingLabels{secretLabelKey: secretLabelValue}); err != nil {
		return v1.Secret{}, fmt.Errorf("unable to fetch Secret provided by KEB for BTP Manager: %w", err)
	}
	switch secretsNum := len(secrets.Items); secretsNum {
	case 1:
		return secrets.Items[0], nil
	default:
		return v1.Secret{}, fmt.Errorf("required number of Secrets for BTP Manager: 1, found: %d", secretsNum)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BtpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BtpOperator{}).
		Owns(&v1.Secret{}).
		Complete(r)
}

func (r *BtpOperatorReconciler) HandleInitialState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	status := cr.GetStatus()
	status.WithState(types.StateProcessing)
	cr.SetStatus(status)
	return r.Status().Update(ctx, cr)
}

func (r *BtpOperatorReconciler) HandleProcessingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)

	r.addTempLabelsToCr(cr)

	installInfo, err := r.getInstallInfo(ctx, cr)
	if err != nil {
		logger.Error(err, "while preparing InstallInfo")
		return err
	}
	if installInfo.ChartPath == "" {
		return fmt.Errorf("no chart path available for processing")
	}

	status := cr.GetStatus()

	ready, err := manifest.InstallChart(&logger, installInfo, []types.ObjectTransform{r.labelTransform})
	if err != nil {
		logger.Error(err, fmt.Sprintf("error while installing resource %s", client.ObjectKeyFromObject(cr)))
		status.WithState(types.StateError)
		cr.SetStatus(status)
		return r.Status().Update(ctx, cr)
	}
	if ready {
		status.WithState(types.StateReady)
		cr.SetStatus(status)
		return r.Status().Update(ctx, cr)
	}

	return nil
}

func (r *BtpOperatorReconciler) addTempLabelsToCr(cr *v1alpha1.BtpOperator) {
	if len(cr.Labels) == 0 {
		cr.Labels = make(map[string]string)
	}
	cr.Labels[labelKeyForChart] = operatorName
}

func (r *BtpOperatorReconciler) getInstallInfo(ctx context.Context, cr *v1alpha1.BtpOperator) (manifest.InstallInfo, error) {
	unstructuredObj := &unstructured.Unstructured{}
	unstructuredBase, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
	if err != nil {
		return manifest.InstallInfo{}, err
	}
	unstructuredObj.Object = unstructuredBase

	installInfo := manifest.InstallInfo{
		ChartInfo: &manifest.ChartInfo{
			ChartPath:   chartPath,
			ReleaseName: cr.GetName(),
			Flags: types.ChartFlags{
				ConfigFlags: types.Flags{
					"Namespace":       chartNamespace,
					"CreateNamespace": true,
					"Wait":            true,
				},
			},
		},
		ResourceInfo: manifest.ResourceInfo{
			BaseResource: unstructuredObj,
		},
		ClusterInfo: custom.ClusterInfo{
			Config: r.Config,
			Client: r.Client,
		},
		Ctx: ctx,
	}

	return installInfo, nil
}

func (r *BtpOperatorReconciler) labelTransform(ctx context.Context, base types.BaseCustomObject, res *types.ManifestResources) error {
	baseLabels := base.GetLabels()
	if _, found := baseLabels[labelKeyForChart]; !found {
		return fmt.Errorf("missing %s label in %s base resource", labelKeyForChart, base.GetName())
	}
	for _, item := range res.Items {
		itemLabels := item.GetLabels()
		if len(itemLabels) == 0 {
			itemLabels = make(map[string]string)
		}
		itemLabels[labelKeyForChart] = baseLabels[labelKeyForChart]
		item.SetLabels(itemLabels)
	}

	return nil
}
