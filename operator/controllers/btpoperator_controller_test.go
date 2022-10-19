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

/*var _ = Describe("deprovisioning - safe delete test", func() {
	It("soft delete should succeed", func() {
		triggerDelete(NewCfg(time.Nanosecond))
		doChecks()
	})

})*/

var _ = Describe("deprovisioning - hard delete test", func() {
	It("hard delete should succeed", func() {

		bindingGvk := schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: btpOperatorServiceBinding}
		instanceGvk := schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: btpOperatorServiceInstance}

		//
		var err error
		err = DeleteDeployment()
		Expect(err).To(BeNil())
		err = DeleteWebhooks()
		Expect(err).To(BeNil())
		err = SoftDelete(bindingGvk)
		Expect(err).To(BeNil())
		err = SoftDelete(instanceGvk)
		Expect(err).To(BeNil())
		//

		triggerDelete(nil)
		doChecks()
	})
})

func doChecks() {
	checkIfNoServicesExists(btpOperatorServiceBindingList)
	checkIfNoServicesExists(btpOperatorServiceInstanceList)
	checkIfNoBtpResourceExists()
}

func triggerDelete(cfgOverride *Cfg) {
	if cfgOverride != nil {
		reconciler.SetupCfg(cfgOverride)
	}

	btpOperator := v1alpha1.BtpOperator{}
	btpOperator.Name = "btpoperator-sample"
	btpOperator.Namespace = "default"
	err := k8sClient.Delete(ctx, &btpOperator)
	Expect(err).To(BeNil())
	time.Sleep(time.Second * 60)
}

func checkIfNoServicesExists(kind string) {
	lst := unstructured.UnstructuredList{}
	lst.SetGroupVersionKind(schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind})
	_ = k8sClient.List(ctx, &lst)
	Expect(lst.Items).To(HaveLen(0))
}

func checkIfNoBtpResourceExists() {
	c, cerr := clientset.NewForConfig(cfg)
	Expect(cerr).To(BeNil())

	_, resourceList, err := c.ServerGroupsAndResources()
	Expect(err).To(BeNil())

	namespaces := &corev1.NamespaceList{}
	if err := k8sClient.List(ctx, namespaces); err != nil {
		return
	}

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
					err = k8sClient.List(ctx, lst, client.InNamespace(namespace.Name), client.MatchingLabels{"managed-by": "btp-operator"})
					if len(lst.Items) > 0 {
						found = true
						break
					}
				} else {
					err = k8sClient.List(ctx, lst, client.MatchingLabels{"managed-by": "btp-operator"})
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

func DeleteDeployment() error {
	deployment := &appsv1.Deployment{}
	deployment.Name = "sap-btp-operator-controller-manager"
	deployment.Namespace = "default"
	if err := k8sClient.Delete(ctx, deployment); err != nil {
		return err
	}
	return nil
}

func SoftDelete(gvk schema.GroupVersionKind) error {
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

func DeleteWebhooks() error {
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

	return nil
}
