package controllers

import (
	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

		createResource(instanceGvk, testNamespace, instanceName)
		ensureResourceExists(instanceGvk)

		createResource(bindingGvk, testNamespace, bindingName)
		ensureResourceExists(bindingGvk)
	})

	It("soft delete (after timeout) should succeed", func() {
		reconciler.SetReconcileConfig(NewReconcileConfig(time.Nanosecond, false))

		triggerDelete()
		doChecks()
	})

	It("soft delete (after hard deletion fail) should succeed", func() {
		reconciler.SetReconcileConfig(NewReconcileConfig(time.Minute*1, true))

		triggerDelete()
		doChecks()
	})

	It("hard delete should succeed", func() {
		reconciler.SetReconcileConfig(NewReconcileConfig(time.Minute*1, false))

		triggerDelete()
		doChecks()
	})
})

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

func triggerDelete() {
	btpOperator := getBtpOperator()
	err := k8sClient.Delete(ctx, &btpOperator)
	Expect(err).To(BeNil())
	time.Sleep(time.Second * 30)
}

func doChecks() {
	checkIfNoServicesExists(btpOperatorServiceBinding)
	checkIfNoServicesExists(btpOperatorServiceInstance)
	checkIfNoBtpResourceExists()
}

func checkIfNoServicesExists(kind string) {
	list := unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind})
	err := k8sClient.List(ctx, &list)
	Expect(errors.IsNotFound(err)).To(BeTrue())
	Expect(list.Items).To(HaveLen(0))
}

func checkIfNoBtpResourceExists() {
	cs, err := clientset.NewForConfig(cfg)
	Expect(err).To(BeNil())

	_, resourceMap, err := cs.ServerGroupsAndResources()
	Expect(err).To(BeNil())

	namespaces := &corev1.NamespaceList{}
	err = k8sClient.List(ctx, namespaces)
	Expect(err).To(BeNil())

	labelMatcher := client.MatchingLabels{labelKey: operatorName}
	fail := false
	for _, resource := range resourceMap {
		gv, _ := schema.ParseGroupVersion(resource.GroupVersion)
		for _, apiResource := range resource.APIResources {
			list := &unstructured.UnstructuredList{}
			gvk := schema.GroupVersionKind{
				Version: gv.Version,
				Group:   gv.Group,
				Kind:    apiResource.Kind,
			}
			list.SetGroupVersionKind(gvk)
			for _, namespace := range namespaces.Items {
				if err := k8sClient.List(ctx, list, client.InNamespace(namespace.Name), labelMatcher); err != nil {
					ignore := errors.IsNotFound(err) || meta.IsNoMatchError(err) || errors.IsMethodNotSupported(err)
					if !ignore {
						fail = true
						break
					}
				} else if len(list.Items) > 0 {
					fail = true
					break
				}
			}
		}
	}
	Expect(fail).To(BeFalse())
}
