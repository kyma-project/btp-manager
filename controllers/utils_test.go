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

	"github.com/kyma-project/btp-manager/internal/conditions"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	btpOperatorName                = "btpoperator"
	btpOperatorKind                = "BtpOperator"
	btpOperatorApiVersion          = `operator.kyma-project.io\v1alpha1`
	secretYamlPath                 = "testdata/test-secret.yaml"
	k8sOpsTimeout                  = time.Second * 3
	k8sOpsPollingInterval          = time.Millisecond * 200
	extraLabelKey                  = "reconciler.kyma-project.io/managed-by"
	extraLabelValue                = "reconciler"
	k8sClientGetPermanentErrMsg    = "expected permanent client.Get error"
	k8sClientGetRetryableErrMsg    = "expected retryable client.Get error"
	k8sClientUpdatePermanentErrMsg = "expected permanent client.Update error"
	k8sClientUpdateRetryableErrMsg = "expected retryable client.Update error"
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

type lazyK8sClient struct {
	client.Client
	requiredRetries int
	getRetries      int
	updateRetries   int
	errorOnGet      bool
	errorOnUpdate   bool
	disableUpdate   bool
}

func newLazyK8sClient(c client.Client, requiredRetries int) *lazyK8sClient {
	return &lazyK8sClient{
		Client:          c,
		requiredRetries: 0,
		getRetries:      0,
		updateRetries:   0,
		errorOnGet:      false,
		errorOnUpdate:   false,
		disableUpdate:   false,
	}
}

func (c *lazyK8sClient) EnableErrorOnGet() {
	c.errorOnGet = true
}

func (c *lazyK8sClient) EnableErrorOnUpdate() {
	c.errorOnUpdate = true
}

func (c *lazyK8sClient) DisableErrorOnGet() {
	c.errorOnGet = false
}

func (c *lazyK8sClient) DisableErrorOnUpdate() {
	c.errorOnUpdate = false
}

func (c *lazyK8sClient) EnableUpdate() {
	c.disableUpdate = false
}

func (c *lazyK8sClient) DisableUpdate() {
	c.disableUpdate = true
}

func (c *lazyK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if c.errorOnGet {
		return errors.New(k8sClientGetPermanentErrMsg)
	}
	if c.getRetries >= c.requiredRetries {
		c.getRetries = 0
		return c.Client.Get(ctx, key, obj, opts...)
	}
	c.getRetries++
	return errors.New(k8sClientGetRetryableErrMsg)
}

func (c *lazyK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.errorOnUpdate {
		return errors.New(k8sClientUpdatePermanentErrMsg)
	}
	if c.disableUpdate {
		return nil
	}
	if c.updateRetries >= c.requiredRetries {
		c.updateRetries = 0
		return c.Client.Update(ctx, obj, opts...)
	}
	c.updateRetries++
	return errors.New(k8sClientUpdateRetryableErrMsg)
}

func (c *lazyK8sClient) Status() client.SubResourceWriter {
	return &fakeSubResourceClient{c}
}

// see fakeSubResourceClient at https://github.com/kubernetes-sigs/controller-runtime/blob/main/pkg/client/fake/client.go
type fakeSubResourceClient struct {
	client.Client
}

func (sw *fakeSubResourceClient) Get(ctx context.Context, obj, subResource client.Object, opts ...client.SubResourceGetOption) error {
	panic("fakeSubResourceClient does not support get")
}

func (sw *fakeSubResourceClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	panic("fakeSubResourceWriter does not support create")
}

func (sw *fakeSubResourceClient) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	updateOptions := client.SubResourceUpdateOptions{}
	updateOptions.ApplyOptions(opts)

	body := obj
	if updateOptions.SubResourceBody != nil {
		body = updateOptions.SubResourceBody
	}
	return sw.Client.Update(ctx, body, &updateOptions.UpdateOptions)
}

func (sw *fakeSubResourceClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	panic("fakeSubResourceWriter does not support patch")
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

func countResourcesWithGivenLabel(gvks []schema.GroupVersionKind, labelKey string, labelValue string) (int, error) {
	var foundResources int
	var ul *unstructured.UnstructuredList
	for _, gvk := range gvks {
		ul = &unstructured.UnstructuredList{}
		ul.SetGroupVersionKind(gvk)
		if err := k8sClient.List(ctx, ul, client.MatchingLabels{labelKey: labelValue}); err != nil {
			return 0, err
		}
		foundResources += len(ul.Items)
	}

	return foundResources, nil
}

func countResourcesForGivenChartVer(gvks []schema.GroupVersionKind, version string) (int, error) {
	return countResourcesWithGivenLabel(gvks, chartVersionLabelKey, version)
}

func copyDirRecursively(src, target string) {
	cmd := exec.Command("cp", "-r", src, target)
	err := cmd.Run()
	Expect(err).To(BeNil())
}

func createPrereqs() error {
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

func isCrNotFound() bool {
	return isNamedCrNotFound(btpOperatorName, kymaNamespace)
}

func isNamedCrNotFound(name, namespace string) bool {
	cr := &v1alpha1.BtpOperator{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cr)
	return k8serrors.IsNotFound(err)
}

func createBtpOperator(name string) *v1alpha1.BtpOperator {
	return &v1alpha1.BtpOperator{
		TypeMeta: metav1.TypeMeta{
			Kind:       btpOperatorKind,
			APIVersion: btpOperatorApiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kymaNamespace,
		},
	}
}

func createDefaultBtpOperator() *v1alpha1.BtpOperator {
	return createBtpOperator(btpOperatorName)
}

func initConfig(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigName,
			Namespace: ChartNamespace,
			Labels:    map[string]string{managedByLabelKey: operatorName},
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

func createChartOrResources(src, dst string, includeWebhooks bool) error {
	Expect(os.MkdirAll(dst, 0700)).To(Succeed())
	src = fmt.Sprintf("%v/.", src)
	cmd := exec.Command("cp", "-r", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copying: %v -> %v\n\nout: %v\nerr: %v", src, dst, string(out), err)
	}

	return filepath.WalkDir(dst, func(path string, de fs.DirEntry, err error) error {
		if !includeWebhooks {
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

func createChartOrResourcesCopyWithoutWebhooksByConfig(src, dst string) error {
	filterWebhooksDisabled := os.Getenv("DISABLE_WEBHOOK_FILTER_FOR_TESTS") == "true"
	return createChartOrResources(src, dst, !filterWebhooksDisabled)
}

func createChartOrResourcesCopyWithoutWebhooks(src, dst string) error {
	return createChartOrResources(src, dst, false)
}

func removeAllFromPath(path string) error {
	return os.RemoveAll(path)
}

func doChecks() {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		defer GinkgoRecover()
		checkIfNoServiceExists(btpOperatorServiceBinding)
	}()
	go func() {
		defer wg.Done()
		checkIfNoBindingSecretExists()
	}()
	go func() {
		defer wg.Done()
		defer GinkgoRecover()
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
	Expect(list.Items).To(HaveLen(0))
	if len(list.Items) == 0 {
		return
	}
	Expect(k8serrors.IsNotFound(err)).To(BeTrue())
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
			err = k8sClient.Update(ctx, &webhook)
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
			err = k8sClient.Update(ctx, &webhook)
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

func getSecretFromNamespace(name, namespace string) *corev1.Secret {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, secret)
	Expect(err).To(BeNil())
	return secret
}

func getSecret(name string) *corev1.Secret {
	return getSecretFromNamespace(name, ChartNamespace)
}

func getOperatorSecret() *corev1.Secret {
	return getSecret(sapBtpServiceOperatorSecretName)
}

func getOperatorConfigMap() *corev1.ConfigMap {
	return getConfigMap(sapBtpServiceOperatorConfigMapName)
}
func getConfigMap(name string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ChartNamespace, Name: name}, configMap)
	Expect(err).To(BeNil())
	return configMap
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
	Eventually(func() error {
		secret := getSecret(CaSecretName)
		ca, ok := secret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
		if !ok || ca == nil {
			return fmt.Errorf("CA bundle not found in secret")
		}

		vw := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
		err := k8sClient.List(ctx, vw, managedByLabelFilter)
		if err != nil {
			return err
		}
		for _, w := range vw.Items {
			for _, n := range w.Webhooks {
				if !bytes.Equal(n.ClientConfig.CABundle, ca) {
					return fmt.Errorf("ValidatingWebhook CABundle does not match CA")
				}
			}
		}

		mw := &admissionregistrationv1.MutatingWebhookConfigurationList{}
		err = k8sClient.List(ctx, mw, managedByLabelFilter)
		if err != nil {
			return err
		}
		for _, w := range mw.Items {
			for _, n := range w.Webhooks {
				if !bytes.Equal(n.ClientConfig.CABundle, ca) {
					return fmt.Errorf("MutatingWebhook CABundle does not match CA")
				}
			}
		}
		return nil
	}).Should(Succeed())
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

func resourceUpdateHandler(old, new any, t string) {
	if cr, ok := new.(*v1alpha1.BtpOperator); ok {
		if oldCr, ok := old.(*v1alpha1.BtpOperator); ok {
			logger.V(1).Info("Triggered update handler for BTPOperator CR", "name", cr.Name, "namespace", cr.Namespace, "action", t, "previous state", oldCr.Status.State, "previous conditions", oldCr.Status.Conditions, "previous version", oldCr.ResourceVersion, "state", cr.Status.State, "conditions", cr.Status.Conditions, "version", cr.ResourceVersion)
		}
		updateCh <- resourceUpdate{Cr: cr, Action: t}
	}
}

func resourceAddDeleteHandler(obj any, t string) {
	if cr, ok := obj.(*v1alpha1.BtpOperator); ok {
		logger.V(1).Info("Triggered add/delete handler for BTPOperator CR", "name", cr.Name, "namespace", cr.Namespace, "action", t, "state", cr.Status.State, "conditions", cr.Status.Conditions, "version", cr.ResourceVersion)
		updateCh <- resourceUpdate{Cr: cr, Action: t}
	}
}

func matchState(state v1alpha1.State) gomegatypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{
		"Action": Equal(resourceUpdated),
		"Cr": PointTo(MatchFields(IgnoreExtras, Fields{
			"Status": MatchFields(IgnoreExtras, Fields{
				"State": Equal(state),
			}),
		})),
	})
}

func matchReadyCondition(state v1alpha1.State, status metav1.ConditionStatus, reason conditions.Reason) gomegatypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{
		"Action": Equal(resourceUpdated),
		"Cr": PointTo(MatchFields(IgnoreExtras, Fields{
			"Status": MatchFields(IgnoreExtras, Fields{
				"State": Equal(state),
				"Conditions": ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(conditions.ReadyType),
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
	*rest.Config
	clientgoappsv1.DeploymentInterface
	Scheme *runtime.Scheme
}

func newDeploymentController(cfg *rest.Config, mgr manager.Manager) controller.Controller {
	appsV1Client, err := v1.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())

	btpOperatorDeploymentReconciler := &deploymentReconciler{
		DeploymentInterface: appsV1Client.Deployments(ChartNamespace),
		Config:              cfg,
		Scheme:              scheme.Scheme,
	}
	deploymentController, err := controller.NewUnmanaged("deployment-controller", controller.Options{
		Reconciler: btpOperatorDeploymentReconciler,
	})
	Expect(err).ToNot(HaveOccurred())

	Expect(deploymentController.Watch(source.Kind(mgr.GetCache(), &appsv1.Deployment{},
		&handler.TypedEnqueueRequestForObject[*appsv1.Deployment]{},
		btpOperatorDeploymentReconciler.watchBtpOperatorDeploymentPredicate()))).
		To(Succeed())

	return deploymentController
}

func (r *deploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	deployment, err := r.Get(ctx, req.NamespacedName.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get deployment")
		return ctrl.Result{}, err
	}
	deploymentProgressingCondition := appsv1.DeploymentCondition{Type: appsv1.DeploymentConditionType(deploymentProgressingConditionType), Status: corev1.ConditionStatus("True")}
	deploymentAvailableCondition := appsv1.DeploymentCondition{Type: appsv1.DeploymentConditionType(deploymentAvailableConditionType), Status: corev1.ConditionStatus("True")}
	conditions := make([]appsv1.DeploymentCondition, 0)
	conditions = append(conditions, deploymentProgressingCondition, deploymentAvailableCondition)
	status := appsv1.DeploymentStatus{Conditions: conditions}
	deployment.Status = status
	_, err = r.UpdateStatus(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "failed to update deployment status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *deploymentReconciler) watchBtpOperatorDeploymentPredicate() predicate.TypedPredicate[*appsv1.Deployment] {
	return predicate.TypedFuncs[*appsv1.Deployment]{
		CreateFunc: func(e event.TypedCreateEvent[*appsv1.Deployment]) bool {
			return true
		},
		UpdateFunc: func(e event.TypedUpdateEvent[*appsv1.Deployment]) bool {
			if len(e.ObjectOld.Status.Conditions) > 0 {
				var progressingConditionStatus, availableConditionStatus string
				for _, c := range e.ObjectOld.Status.Conditions {
					if string(c.Type) == deploymentProgressingConditionType {
						progressingConditionStatus = string(c.Status)
					} else if string(c.Type) == deploymentAvailableConditionType {
						availableConditionStatus = string(c.Status)
					}
				}
				if progressingConditionStatus != "True" || availableConditionStatus != "True" {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(e event.TypedDeleteEvent[*appsv1.Deployment]) bool {
			return false
		},
		GenericFunc: func(e event.TypedGenericEvent[*appsv1.Deployment]) bool {
			return false
		},
	}
}
