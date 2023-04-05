package controllers

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	"github.com/kyma-project/module-manager/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	btpOperatorName       = "btp-operator-test"
	btpOperatorKind       = "BtpOperator"
	btpOperatorApiVersion = `operator.kyma-project.io\v1alpha1`
	secretYamlPath        = "testdata/test-secret.yaml"
	priorityClassYamlPath = "testdata/test-priorityclass.yaml"
	k8sOpsTimeout         = time.Second * 3
	k8sOpsPollingInterval = time.Millisecond * 200
)

// fake K8s clients with overridden behavior
type timeoutK8sClient struct {
	client.Client
}

func newTimeoutK8sClient(c client.Client) *timeoutK8sClient {
	return &timeoutK8sClient{c}
}

func (c *timeoutK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == instanceGvk.Kind || kind == bindingGvk.Kind {
		deleteAllOfCtx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
		defer cancel()
		return c.Client.DeleteAllOf(deleteAllOfCtx, obj, opts...)
	}

	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

type errorK8sClient struct {
	client.Client
}

func newErrorK8sClient(c client.Client) *errorK8sClient {
	return &errorK8sClient{c}
}

func (c *errorK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == instanceGvk.Kind || kind == bindingGvk.Kind {
		deleteAllOfCtx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
		defer cancel()
		_ = c.Client.DeleteAllOf(deleteAllOfCtx, obj, opts...)
		return errors.New("expected DeleteAllOf error")
	}

	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

// module-resources paths
func getApplyPath() string {
	return fmt.Sprintf("%s%capply", ResourcesPath, os.PathSeparator)
}

func getDeletePath() string {
	return fmt.Sprintf("%s%cdelete", ResourcesPath, os.PathSeparator)
}

func getToDeleteYamlPath() string {
	return fmt.Sprintf("%s%cto-delete.yml", getDeletePath(), os.PathSeparator)
}

func getTempPath() string {
	return fmt.Sprintf("%s%ctemp", ResourcesPath, os.PathSeparator)
}

func assertResourcesExistence(uns ...*unstructured.Unstructured) {
	for _, u := range uns {
		gvk := u.GroupVersionKind()
		temp := &unstructured.Unstructured{}
		temp.SetGroupVersionKind(gvk)
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, temp)).To(Succeed())
	}
}

func assertResourcesRemoval(uns ...*unstructured.Unstructured) {
	for _, u := range uns {
		gvk := u.GroupVersionKind()
		temp := &unstructured.Unstructured{}
		temp.SetGroupVersionKind(gvk)
		gr := schema.GroupResource{
			Group:    gvk.Group,
			Resource: fmt.Sprintf("%ss", strings.ToLower(gvk.Kind)),
		}
		expectedErr := k8serrors.NewNotFound(gr, u.GetName())
		expectedErr.ErrStatus.TypeMeta.APIVersion = "v1"
		expectedErr.ErrStatus.TypeMeta.Kind = "Status"
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, temp)).To(MatchError(expectedErr))
	}
}

func moveOrCopyNFilesFromDirToDir(filesNum int, deleteFiles bool, srcDir, targetDir string) error {
	if err := os.Mkdir(targetDir, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	files, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for i, f := range files {
		if i >= filesNum {
			break
		}
		input, err := os.ReadFile(fmt.Sprintf("%s%c%s", srcDir, os.PathSeparator, f.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(fmt.Sprintf("%s%c%s", targetDir, os.PathSeparator, f.Name()), input, 0700); err != nil {
			return err
		}
		if deleteFiles {
			if err := os.Remove(fmt.Sprintf("%s%c%s", srcDir, os.PathSeparator, f.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func getUniqueGvksFromObjects(objs []runtime.Object) []schema.GroupVersionKind {
	gvks := make([]schema.GroupVersionKind, 0)
	helper := make(map[string]struct{}, 0)
	for _, o := range objs {
		gvk := o.GetObjectKind().GroupVersionKind()
		gvkString := gvk.String()
		if _, exists := helper[gvkString]; exists {
			continue
		}
		helper[gvkString] = struct{}{}
		gvks = append(gvks, gvk)
	}

	return gvks
}

func countResourcesForGivenChartVer(gvks []schema.GroupVersionKind, version string) (int, error) {
	var foundResources int
	var ul *unstructured.UnstructuredList
	for _, gvk := range gvks {
		ul = &unstructured.UnstructuredList{}
		ul.SetGroupVersionKind(gvk)
		if err := k8sClient.List(ctx, ul, client.MatchingLabels{chartVersionKey: version}); err != nil {
			return 0, err
		}
		foundResources += len(ul.Items)
	}

	return foundResources, nil
}

func copyDirRecursively(src, target string) {
	cmd := exec.Command("cp", "-r", src, target)
	err := cmd.Run()
	Expect(err).To(BeNil())
}

func createPrereqs() error {
	pClass := &schedulingv1.PriorityClass{}
	Expect(createK8sResourceFromYaml(pClass, priorityClassYamlPath)).To(Succeed())
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(pClass), pClass); err != nil {
		if k8serrors.IsNotFound(err) {
			Eventually(func() error { return k8sClient.Create(ctx, pClass) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		} else {
			return err
		}
	}

	kymaNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: kymaNamespace}}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kymaNs), kymaNs); err != nil {
		if k8serrors.IsNotFound(err) {
			Eventually(func() error { return k8sClient.Create(ctx, kymaNs) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		} else {
			return err
		}
	}

	return nil
}

func setFinalizers(resource *unstructured.Unstructured) {
	finalizers := []string{"test-finalizer"}
	Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.GetName()}, resource)).To(Succeed())
	Expect(unstructured.SetNestedStringSlice(resource.Object, finalizers, "metadata", "finalizers")).To(Succeed())
	Expect(k8sClient.Update(ctx, resource)).To(Succeed())
}

func getCurrentCrState() types.State {
	cr := &v1alpha1.BtpOperator{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr); err != nil {
		return ""
	}
	return cr.GetStatus().State
}

func getCurrentCrStatus() types.Status {
	cr := &v1alpha1.BtpOperator{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr); err != nil {
		return types.Status{}
	}
	GinkgoLogr.Info(fmt.Sprintf("Got CR status: %s\n", cr.Status.State))
	return cr.GetStatus()
}

func isCrNotFound() bool {
	cr := &v1alpha1.BtpOperator{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
	return k8serrors.IsNotFound(err)
}

func createBtpOperator() *v1alpha1.BtpOperator {
	return &v1alpha1.BtpOperator{
		TypeMeta: metav1.TypeMeta{
			Kind:       btpOperatorKind,
			APIVersion: btpOperatorApiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      btpOperatorName,
			Namespace: defaultNamespace,
		},
	}
}

func initConfig(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigName,
			Namespace: ChartNamespace,
		},
		Data: data,
	}
}

func createCorrectSecretFromYaml() (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	data, err := os.ReadFile(secretYamlPath)
	if err != nil {
		return nil, fmt.Errorf("while reading the required Secret YAML: %w", err)
	}
	err = yaml.Unmarshal(data, secret)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling Secret YAML to struct: %w", err)
	}

	return secret, nil
}

func createSecretWithoutKeys() (*corev1.Secret, error) {
	secret, err := createCorrectSecretFromYaml()
	if err != nil {
		return nil, fmt.Errorf("while creating Secret from YAML: %w", err)
	}
	delete(secret.Data, "cluster_id")
	delete(secret.Data, "clientsecret")

	return secret, nil
}

func createSecretWithoutValues() (*corev1.Secret, error) {
	secret, err := createCorrectSecretFromYaml()
	if err != nil {
		return nil, fmt.Errorf("while creating Secret from YAML: %w", err)
	}
	secret.Data["cluster_id"] = []byte("")
	secret.Data["clientsecret"] = []byte("")

	return secret, nil
}

func createK8sResourceFromYaml[T runtime.Object](resource T, yamlPath string) error {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("while reading YAML: %w", err)
	}
	err = yaml.Unmarshal(data, resource)
	if err != nil {
		return fmt.Errorf("while unmarshalling YAML to struct: %w", err)
	}

	return nil
}

func ensureResourceExists(gvk schema.GroupVersionKind) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	err := k8sClient.List(ctx, list)
	Expect(err).To(BeNil())
	Expect(list.Items).To(HaveLen(1))
}

func createResource(gvk schema.GroupVersionKind, namespace string, name string) *unstructured.Unstructured {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	object.SetNamespace(namespace)
	object.SetName(name)
	kind := object.GetObjectKind().GroupVersionKind().Kind
	if kind == instanceGvk.Kind {
		populateServiceInstanceFields(object)
	} else if kind == bindingGvk.Kind {
		populateServiceBindingFields(object)
	}
	Expect(k8sClient.Create(ctx, object)).To(BeNil())

	return object
}

func populateServiceInstanceFields(object *unstructured.Unstructured) {
	Expect(unstructured.SetNestedField(object.Object, "test-service", "spec", "serviceOfferingName")).To(Succeed())
	Expect(unstructured.SetNestedField(object.Object, "test-plan", "spec", "servicePlanName")).To(Succeed())
	Expect(unstructured.SetNestedField(object.Object, "test-service-instance-external", "spec", "externalName")).To(Succeed())
}

func populateServiceBindingFields(object *unstructured.Unstructured) {
	Expect(unstructured.SetNestedField(object.Object, "test-service-instance", "spec", "serviceInstanceName")).To(Succeed())
	Expect(unstructured.SetNestedField(object.Object, "test-binding-external", "spec", "externalName")).To(Succeed())
	Expect(unstructured.SetNestedField(object.Object, "test-service-binding-secret", "spec", "secretName")).To(Succeed())
}

func filterWebhooks(file []byte) (filtered []byte, hasWebhook bool) {
	lines := strings.Split(string(file), "\n")
	var buffer []byte
	isWebhook := false
	for _, l := range lines {
		buffer = append(buffer, []byte(l+"\n")...)
		if l == "---" && len(buffer) != 0 {
			if isWebhook {
				split := strings.Split(string(buffer), "\n")
				// hack for one case where helm templating block spans across two adjacent documents
				for _, spl := range split {
					splTrunc := strings.ReplaceAll(spl, " ", "")
					splTrunc = strings.ReplaceAll(splTrunc, "-", "")
					if splTrunc != "{{end}}" {
						break
					}
					filtered = append(filtered, []byte(spl+"\n")...)
				}
			} else {
				filtered = append(filtered, buffer...)
			}
			buffer = []byte{}
			isWebhook = false
		}
		if strings.HasPrefix(l, "kind: ") {
			split := strings.Split(l, ":")
			if len(split) != 2 {
				continue
			}
			kind := strings.TrimLeft(split[1], " ")
			if kind == "MutatingWebhookConfiguration" || kind == "ValidatingWebhookConfiguration" {
				isWebhook, hasWebhook = true, true
			}
		}
	}
	if !isWebhook {
		filtered = append(filtered, buffer...)
	}
	return
}

func createChartOrResourcesCopyWithoutWebhooks(src, dst string) error {
	Expect(os.MkdirAll(dst, 0700)).To(Succeed())
	src = fmt.Sprintf("%v/.", src)
	cmd := exec.Command("cp", "-r", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copying: %v -> %v\n\nout: %v\nerr: %v", src, dst, string(out), err)
	}
	filterWebhooksDisabled := os.Getenv("DISABLE_WEBHOOK_FILTER_FOR_TESTS")

	return filepath.WalkDir(dst, func(path string, de fs.DirEntry, err error) error {
		if filterWebhooksDisabled == "true" {
			return nil
		}
		if de.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		dat, err := os.ReadFile(path)
		Expect(err).To(BeNil())

		documents, filtered := filterWebhooks(dat)
		if len(documents) == 0 {
			Expect(os.Remove(path)).To(Succeed())
		}
		if filtered {
			Expect(os.WriteFile(path, documents, 0700)).To(Succeed())
		}
		return nil
	})
}

func removeAllFromPath(path string) error {
	return os.RemoveAll(path)
}

func doChecks() {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		checkIfNoServiceExists(btpOperatorServiceBinding)
	}()
	go func() {
		defer wg.Done()
		checkIfNoBindingSecretExists()
	}()
	go func() {
		defer wg.Done()
		checkIfNoServiceExists(btpOperatorServiceInstance)
	}()
	go func() {
		defer wg.Done()
		checkIfNoBtpResourceExists()
	}()
	wg.Wait()
}

func checkIfNoServiceExists(kind string) {
	list := unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind})
	err := k8sClient.List(ctx, &list)
	Expect(k8serrors.IsNotFound(err)).To(BeTrue())
	Expect(list.Items).To(HaveLen(0))
}

func checkIfNoBindingSecretExists() {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: bindingName, Namespace: ChartNamespace}, secret)
	Expect(*secret).To(BeEquivalentTo(corev1.Secret{}))
	Expect(k8serrors.IsNotFound(err)).To(BeTrue())
}

func checkIfNoBtpResourceExists() {
	gvks, err := ymlutils.GatherChartGvks(ChartPath)
	Expect(err).To(BeNil())

	found := false
	for _, gvk := range gvks {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Version: gvk.Version,
			Group:   gvk.Group,
			Kind:    gvk.Kind,
		})
		if err := k8sClient.List(ctx, list, managedByLabelFilter); err != nil {
			if !canIgnoreErr(err) {
				found = true
				break
			}
		} else if len(list.Items) > 0 {
			found = true
			break
		}
	}
	Expect(found).To(BeFalse())
}

func canIgnoreErr(err error) bool {
	return k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) || k8serrors.IsMethodNotSupported(err)
}

func replaceCaBundleInValidatingWebhooks(newCaBundle []byte) bool {
	webhookConfig := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
	err := k8sClient.List(ctx, webhookConfig, managedByLabelFilter)
	Expect(err).To(BeNil())
	Expect(len(webhookConfig.Items) > 0).To(BeTrue())
	if len(webhookConfig.Items) > 0 {
		webhook := webhookConfig.Items[0]
		Expect(len(webhook.Webhooks) > 0).To(BeTrue())
		if len(webhook.Webhooks) > 0 {
			webhook.Webhooks[0].ClientConfig.CABundle = newCaBundle
			err := k8sClient.Update(ctx, &webhook)
			Expect(err).To(BeNil())
			return true
		}
	}
	return false
}

func replaceCaBundleInMutatingWebhooks(newCaBundle []byte) bool {
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfigurationList{}
	err := k8sClient.List(ctx, webhookConfig, managedByLabelFilter)
	Expect(err).To(BeNil())
	Expect(len(webhookConfig.Items) > 0).To(BeTrue())
	if len(webhookConfig.Items) > 0 {
		webhook := webhookConfig.Items[0]
		Expect(len(webhook.Webhooks) > 0).To(BeTrue())
		if len(webhook.Webhooks) > 0 {
			webhook.Webhooks[0].ClientConfig.CABundle = newCaBundle
			err := k8sClient.Update(ctx, &webhook)
			Expect(err).To(BeNil())
			return true
		}
	}
	return false
}

func replaceSecretData(secret *corev1.Secret, key string, value []byte, key2 string, value2 []byte) {
	data := secret.Data
	if key != "" && value != nil {
		data[key] = value
	}

	if key2 != "" && value2 != nil {
		data[key2] = value2
	}
	secret.Data = data
	err := k8sClient.Update(ctx, secret)
	Expect(err).To(BeNil())
}

func getSecret(name string) *corev1.Secret {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ChartNamespace, Name: name}, secret)
	Expect(err).To(BeNil())
	return secret
}

func checkHowManySecondsToExpiration(name string) float64 {
	data, err := reconciler.getDataFromSecret(ctx, name)
	Expect(err).To(BeNil())
	key, err := reconciler.mapSecretNameToSecretDataKey(name)
	Expect(err).To(BeNil())
	value, err := reconciler.getValueByKey(reconciler.buildKeyNameWithExtension(key, CertificatePostfix), data)
	Expect(err).To(BeNil())
	decoded, _ := pem.Decode(value)
	cert, err := x509.ParseCertificate(decoded.Bytes)
	Expect(err).To(BeNil())
	diff := cert.NotAfter.Sub(time.Now())
	return diff.Seconds()
}

func ensureAllWebhooksManagedByBtpOperatorHaveCorrectCABundles() {
	secret := getSecret(CaSecret)
	ca, ok := secret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
	Expect(ok).To(BeTrue())
	Expect(ca).To(Not(BeNil()))
	vw := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
	err := k8sClient.List(ctx, vw, managedByLabelFilter)
	Expect(err).To(BeNil())
	for _, w := range vw.Items {
		for _, n := range w.Webhooks {
			Expect(bytes.Equal(n.ClientConfig.CABundle, ca)).To(BeTrue())
		}
	}
	mw := &admissionregistrationv1.MutatingWebhookConfigurationList{}
	err = k8sClient.List(ctx, mw, managedByLabelFilter)
	Expect(err).To(BeNil())
	for _, w := range mw.Items {
		for _, n := range w.Webhooks {
			Expect(bytes.Equal(n.ClientConfig.CABundle, ca)).To(BeTrue())
		}
	}
}

func structToByteArray(s any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(s)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

type resourceUpdate struct {
	Cr     *v1alpha1.BtpOperator
	Action string
}

func resourceUpdateHandler(obj any, t string) {
	if cr, ok := obj.(*v1alpha1.BtpOperator); ok {
		logger.V(1).Info("Triggered update handler for BTPOperator CR", "name", cr.Name, "action", t, "state", cr.Status.State, "conditions", cr.Status.Conditions)
		updateCh <- resourceUpdate{Cr: cr, Action: t}
	}
}

func matchState(state types.State) gomegatypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{
		"Action": Equal(resourceUpdated),
		"Cr": PointTo(MatchFields(IgnoreExtras, Fields{
			"Status": MatchFields(IgnoreExtras, Fields{
				"State": Equal(state),
			}),
		})),
	})
}

func matchReadyCondition(state types.State, status metav1.ConditionStatus, reason Reason) gomegatypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{
		"Action": Equal(resourceUpdated),
		"Cr": PointTo(MatchFields(IgnoreExtras, Fields{
			"Status": MatchFields(IgnoreExtras, Fields{
				"State": Equal(state),
				"Conditions": ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(ReadyType),
					"Reason": Equal(string(reason)),
					"Status": Equal(status),
				}))),
			}),
		})),
	})
}

func matchDeleted() gomegatypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{"Action": Equal(resourceDeleted)})
}

type deploymentReconciler struct {
	client.Client
	*rest.Config
	Scheme *runtime.Scheme
}

func (r *deploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	deployment := &v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, deployment); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get deployment")
		return ctrl.Result{}, err
	}
	var progressingConditionStatus, availableConditionStatus string
	if len(deployment.Status.Conditions) > 0 {
		for _, c := range deployment.Status.Conditions {
			if string(c.Type) == deploymentProgressingConditionType {
				progressingConditionStatus = string(c.Status)
			} else if string(c.Type) == deploymentAvailableConditionType {
				availableConditionStatus = string(c.Status)
			}
		}
		if progressingConditionStatus == "True" && availableConditionStatus == "True" {
			return ctrl.Result{}, nil
		}
	}
	deploymentProgressingCondition := v1.DeploymentCondition{Type: deploymentProgressingConditionType, Status: "True"}
	deploymentAvailableCondition := v1.DeploymentCondition{Type: deploymentAvailableConditionType, Status: "True"}
	deployment.Status.Conditions = append(deployment.Status.Conditions, deploymentProgressingCondition, deploymentAvailableCondition)
	return ctrl.Result{}, r.Update(ctx, deployment)
}

func (r *deploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Deployment{},
			builder.WithPredicates(r.watchBtpOperatorDeploymentPredicate())).
		Complete(r)
}

func (r *deploymentReconciler) watchBtpOperatorDeploymentPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if e.Object.GetName() == DeploymentName {
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectNew.GetName() == DeploymentName {
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}
