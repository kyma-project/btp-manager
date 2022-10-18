package controllers

import (
	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("deprovisioning - safe delete test", func() {
	It("soft delete should succeed", func() {
		triggerDelete(NewCfg(time.Nanosecond))
		doChecks()
	})

})

var _ = Describe("deprovisioning - hard delete test", func() {
	It("hard delete should succeed", func() {
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
