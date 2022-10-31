package controllers

import (
	"context"
	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	testOperatorName = "btp-operator-tests"
	testNamespace    = "default"
	instanceName     = "my-service-instance"
	bindingName      = "my-binding"
)

var _ = Describe("provisioning test within service instances and bindings", func() {
	BeforeEach(func() {
		btpOperator := getBtpOperator()

		err := k8sClient.Create(ctx, &btpOperator)
		Expect(err).To(BeNil())

		time.Sleep(time.Second * 30)

		ns := &corev1.Namespace{}
		ns.Name = "alpha"
		err = k8sClient.Create(context.Background(), ns)
		Expect(err).To(BeNil())
		ns = &corev1.Namespace{}
		ns.Name = "beta"
		err = k8sClient.Create(context.Background(), ns)
		Expect(err).To(BeNil())

		createResourcesOtherThanBtpOperatorRelated()

		err = clearWebhooks()
		Expect(err).To(BeNil())

		createResource(instanceGvk, testNamespace, instanceName)
		ensureResourceExists(instanceGvk)

		createResource(bindingGvk, testNamespace, bindingName)
		ensureResourceExists(bindingGvk)
	})

	It("soft delete (after timeout) should succeed", func() {
		reconciler.SetReconcileConfig(NewReconcileConfig(time.Second, testScenarioWithTimeout))

		triggerDelete()
		doChecks()
	})

	It("soft delete (after hard deletion fail) should succeed", func() {
		reconciler.SetReconcileConfig(NewReconcileConfig(time.Minute*1, testScenarioWithError))

		triggerDelete()
		doChecks()
	})

	It("hard delete should succeed", func() {
		reconciler.SetReconcileConfig(NewReconcileConfig(time.Minute*1, ""))

		triggerDelete()
		doChecks()
	})
})

func createResourcesOtherThanBtpOperatorRelated() {
	deployment, replicaSet, secret, configMap := getResourcesOtherThanBtpOperatorRelated()

	err := k8sClient.Create(context.Background(), deployment)
	Expect(err).To(BeNil())
	err = k8sClient.Create(context.Background(), replicaSet)
	Expect(err).To(BeNil())
	err = k8sClient.Create(context.Background(), secret)
	Expect(err).To(BeNil())
	err = k8sClient.Create(context.Background(), configMap)
	Expect(err).To(BeNil())
}

func ensureResourcesOtherThanBtpOperatorRelatedExists() {
	deploymentKey, replicaSetKey, secretKey, configMapKey := getResourcesOtherThanBtpOperatorRelated()

	deploymentGet := &v1.Deployment{}
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deploymentKey), deploymentGet)
	Expect(err).To(BeNil())
	Expect(deploymentGet.UID).ToNot(BeEmpty())

	replicaSetGet := &v1.ReplicaSet{}
	err = k8sClient.Get(ctx, client.ObjectKeyFromObject(replicaSetKey), replicaSetGet)
	Expect(err).To(BeNil())
	Expect(replicaSetGet.UID).ToNot(BeEmpty())

	secretGet := &corev1.Secret{}
	err = k8sClient.Get(ctx, client.ObjectKeyFromObject(secretKey), secretGet)
	Expect(err).To(BeNil())
	Expect(secretGet.UID).ToNot(BeEmpty())

	configMapGet := &corev1.ConfigMap{}
	err = k8sClient.Get(ctx, client.ObjectKeyFromObject(configMapKey), configMapGet)
	Expect(err).To(BeNil())
	Expect(configMapGet.UID).ToNot(BeEmpty())
}

func getResourcesOtherThanBtpOperatorRelated() (*v1.Deployment, *v1.ReplicaSet, *corev1.Secret, *corev1.ConfigMap) {
	deployment := &v1.Deployment{}
	deployment.Name = "alpha-deployment"
	deployment.Namespace = "alpha"

	replicaSet := &v1.ReplicaSet{}
	replicaSet.Name = "beta-replicaset"
	replicaSet.Namespace = "beta"

	secret := &corev1.Secret{}
	secret.Name = "alpha-secret"
	secret.Namespace = "alpha"

	configMap := &corev1.ConfigMap{}
	configMap.Name = "beta-configmap"
	configMap.Namespace = "beta"

	return deployment, replicaSet, secret, configMap
}

func getBtpOperator() v1alpha1.BtpOperator {
	btpOperator := v1alpha1.BtpOperator{}
	btpOperator.Name = testOperatorName
	btpOperator.Namespace = testNamespace
	return btpOperator
}

func ensureResourceExists(gvk schema.GroupVersionKind) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	err := k8sClient.List(ctx, list)
	Expect(err).To(BeNil())
	Expect(list.Items).To(HaveLen(1))
}

func createResource(gvk schema.GroupVersionKind, namespace string, name string) {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	object.SetNamespace(namespace)
	object.SetName(name)
	err := k8sClient.Create(ctx, object)
	Expect(err).To(BeNil())
}

func clearWebhooks() error {
	mutatingWebhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err := k8sClient.DeleteAllOf(ctx, mutatingWebhook, labelFilter); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := k8sClient.DeleteAllOf(ctx, validatingWebhook, labelFilter); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func triggerDelete() {
	btpOperator := getBtpOperator()
	err := k8sClient.Delete(ctx, &btpOperator)
	Expect(err).To(BeNil())
	time.Sleep(time.Second * 30)
}

func doChecks() {
	checkIfNoServicesExists(btpOperatorServiceBinding)
	checkIfNoBindingSecretExists()
	checkIfNoServicesExists(btpOperatorServiceInstance)
	checkIfNoBtpResourceExists()
	ensureResourcesOtherThanBtpOperatorRelatedExists()
}

func checkIfNoServicesExists(kind string) {
	list := unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind})
	err := k8sClient.List(ctx, &list)
	Expect(errors.IsNotFound(err)).To(BeTrue())
	Expect(list.Items).To(HaveLen(0))
}

func checkIfNoBindingSecretExists() {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: bindingName, Namespace: testNamespace}, secret)
	Expect(*secret).To(BeEquivalentTo(corev1.Secret{}))
	Expect(errors.IsNotFound(err)).To(BeTrue())
}

func checkIfNoBtpResourceExists() {
	cs, err := clientset.NewForConfig(cfg)
	Expect(err).To(BeNil())

	_, resourceMap, err := cs.ServerGroupsAndResources()
	Expect(err).To(BeNil())

	namespaces := &corev1.NamespaceList{}
	err = k8sClient.List(ctx, namespaces)
	Expect(err).To(BeNil())

	found := false
	for _, resource := range resourceMap {
		gv, _ := schema.ParseGroupVersion(resource.GroupVersion)
		for _, apiResource := range resource.APIResources {
			list := &unstructured.UnstructuredList{}
			list.SetGroupVersionKind(schema.GroupVersionKind{
				Version: gv.Version,
				Group:   gv.Group,
				Kind:    apiResource.Kind,
			})
			for _, namespace := range namespaces.Items {
				if err := k8sClient.List(ctx, list, client.InNamespace(namespace.Name), labelFilter); err != nil {
					ignore := errors.IsNotFound(err) || meta.IsNoMatchError(err) || errors.IsMethodNotSupported(err)
					if !ignore {
						found = true
						break
					}
				} else if len(list.Items) > 0 {
					found = true
					break
				}
			}
		}
	}
	Expect(found).To(BeFalse())
}
