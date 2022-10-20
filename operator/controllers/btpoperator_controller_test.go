package controllers

import (
	"fmt"
	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	btpOperatorGroup               = "services.cloud.sap.com"
	btpOperatorApiVer              = "v1"
	btpOperatorApiVerBadOne        = "v2"
	btpOperatorServiceInstance     = "ServiceInstance"
	btpOperatorServiceInstanceList = btpOperatorServiceInstance + "List"
	btpOperatorServiceBinding      = "ServiceBinding"
	btpOperatorServiceBindingList  = btpOperatorServiceBinding + "List"
)

func PreHardDelete() error {
	deployment := &appsv1.Deployment{}
	deployment.Name = "sap-btp-operator-controller-manager"
	deployment.Namespace = "default"
	if err := k8sClient.Delete(ctx, deployment); err != nil {
		return err
	}

	mutatingWebHook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	mutatingWebHook.Name = "sap-btp-operator-mutating-webhook-configuration"

	if err := k8sClient.Delete(ctx, mutatingWebHook); err != nil {
		return err
	}

	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	validatingWebhook.Name = "sap-btp-operator-validating-webhook-configuration"

	if err := k8sClient.Delete(ctx, validatingWebhook); err != nil {
		return err
	}

	x := func(gvk schema.GroupVersionKind) error {
		errs := fmt.Errorf("")
		listGvk := gvk
		listGvk.Kind = gvk.Kind + "List"
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(listGvk)
		if errs := k8sClient.List(ctx, list); errs != nil {
			errs = fmt.Errorf("%w; could not list in soft delete", errs)
			return errs
		}

		for _, item := range list.Items {
			item.SetFinalizers([]string{})
			if err := k8sClient.Update(ctx, &item); err != nil {
				errs = fmt.Errorf("%w; error occurred in soft delete when updating btp operator resources", err)
			}
		}

		if errs != nil && len(errs.Error()) > 0 {
			return errs
		} else {
			return nil
		}
	}

	bindingGvk := schema.GroupVersionKind{
		Group:   btpOperatorGroup,
		Version: btpOperatorApiVer,
		Kind:    btpOperatorServiceBinding,
	}
	if err := x(bindingGvk); err != nil {
		return err
	}

	instanceGvk := schema.GroupVersionKind{
		Group:   btpOperatorGroup,
		Version: btpOperatorApiVer,
		Kind:    btpOperatorServiceInstance,
	}
	if err := x(instanceGvk); err != nil {
		return err
	}

	return nil
}

var _ = Describe("provisioning test within service instances and bindings", func() {
	BeforeEach(func() {
		btpOperator := &v1alpha1.BtpOperator{}
		btpOperator.Namespace = "default"
		btpOperator.Name = "btpoperator-sample"

		var err error
		err = k8sClient.Create(ctx, btpOperator)
		Expect(err).To(BeNil())

		time.Sleep(time.Second * 20)

		instanceGvk := schema.GroupVersionKind{
			Group:   btpOperatorGroup,
			Version: btpOperatorApiVer,
			Kind:    btpOperatorServiceInstance,
		}

		createResource(instanceGvk, "default", "my-service-instance")
		ensureResourceExists(instanceGvk)

		bindingGvk := schema.GroupVersionKind{
			Group:   btpOperatorGroup,
			Version: btpOperatorApiVer,
			Kind:    btpOperatorServiceBinding,
		}

		createResource(bindingGvk, "default", "my-binding")
		ensureResourceExists(bindingGvk)
	})

	It("soft delete (after timeout) should succeed", func() {
		cfg := NewReconcileConfig(btpOperatorApiVer, btpOperatorGroup, btpOperatorServiceBinding, btpOperatorServiceInstance, time.Nanosecond, false)
		reconciler.SetReconcileConfig(cfg)

		triggerDelete()
		doChecks()
	})

	It("soft delete (after hard deletion fail) should succeed", func() {
		cfg := NewReconcileConfig(btpOperatorApiVer, btpOperatorGroup, btpOperatorServiceBinding, btpOperatorServiceInstance, time.Minute*1, true)
		reconciler.SetReconcileConfig(cfg)

		triggerDelete()
		doChecks()
	})

	It("hard delete should succeed", func() {
		cfg := NewReconcileConfig(btpOperatorApiVer, btpOperatorGroup, btpOperatorServiceBinding, btpOperatorServiceInstance, time.Minute*1, false)
		reconciler.SetReconcileConfig(cfg)

		PreHardDelete()
		triggerDelete()
		doChecks()
	})
})

func ensureResourceExists(gvk schema.GroupVersionKind) {
	lst := &unstructured.UnstructuredList{}
	lst.SetGroupVersionKind(gvk)
	err := k8sClient.List(ctx, lst)
	Expect(err).To(BeNil())
	Expect(lst.Items).To(HaveLen(1))
}

func createResource(gvk schema.GroupVersionKind, namespace string, name string) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetNamespace(namespace)
	obj.SetName(name)
	err := k8sClient.Create(ctx, obj)
	Expect(err).To(BeNil())
}

func triggerDelete() {
	btpOperator := v1alpha1.BtpOperator{}
	btpOperator.Name = "btpoperator-sample"
	btpOperator.Namespace = "default"
	err := k8sClient.Delete(ctx, &btpOperator)
	Expect(err).To(BeNil())
	time.Sleep(time.Second * 120)
}

func doChecks() {
	checkIfNoServicesExists(btpOperatorServiceBindingList)
	checkIfNoServicesExists(btpOperatorServiceInstanceList)
	checkIfNoBtpResourceExists()
}

func checkIfNoServicesExists(kind string) {
	lst := unstructured.UnstructuredList{}
	lst.SetGroupVersionKind(schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind})
	_ = k8sClient.List(ctx, &lst)
	Expect(lst.Items).To(HaveLen(0))
}

func checkIfNoBtpResourceExists() {
	c, err1 := clientset.NewForConfig(cfg)
	Expect(err1).To(BeNil())

	_, resourceList, err2 := c.ServerGroupsAndResources()
	Expect(err2).To(BeNil())

	namespaces := &corev1.NamespaceList{}
	err3 := k8sClient.List(ctx, namespaces)
	Expect(err3).To(BeNil())

	var found bool
	for _, resource := range resourceList {
		gv, _ := schema.ParseGroupVersion(resource.GroupVersion)
		for _, nestedResource := range resource.APIResources {
			lst := &unstructured.UnstructuredList{}
			lst.SetGroupVersionKind(schema.GroupVersionKind{
				Version: gv.Version,
				Group:   gv.Group,
				Kind:    nestedResource.Kind + "List",
			})

			for _, namespace := range namespaces.Items {
				if nestedResource.Namespaced {
					_ = k8sClient.List(ctx, lst, client.InNamespace(namespace.Name), client.MatchingLabels{"managed-by": "btp-operator"})
					if len(lst.Items) > 0 {
						found = true
						break
					}
				} else {
					_ = k8sClient.List(ctx, lst, client.MatchingLabels{"managed-by": "btp-operator"})
					if len(lst.Items) > 0 {
						found = true
						break
					}
				}
			}
		}
	}

	Expect(found).To(BeFalse())
}
