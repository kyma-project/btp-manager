package config_test

import (
	"context"

	"github.com/kyma-project/btp-manager/controllers/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ = Describe("OperatorConfigHandler", func() {
	const (
		operatorConfigName    = "sap-btp-operator-config"
		enableLimitedCacheKey = "ENABLE_LIMITED_CACHE"
		operatorNamespace     = "kyma-system"
	)

	var (
		handler    *config.OperatorConfigHandler
		fakeClient client.Client
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		handler = config.NewOperatorConfigHandler(fakeClient)
	})

	Describe("Predicates", func() {
		var operatorCM *corev1.ConfigMap

		BeforeEach(func() {
			operatorCM = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      operatorConfigName,
					Namespace: operatorNamespace,
				},
				Data: map[string]string{enableLimitedCacheKey: "true"},
			}
		})

		It("UpdateFunc returns true when ENABLE_LIMITED_CACHE changes", func() {
			updated := operatorCM.DeepCopy()
			updated.Data[enableLimitedCacheKey] = "false"
			result := handler.Predicates().UpdateFunc(event.UpdateEvent{
				ObjectOld: operatorCM,
				ObjectNew: updated,
			})
			Expect(result).To(BeTrue())
		})

		It("UpdateFunc returns false when ENABLE_LIMITED_CACHE does not change", func() {
			updated := operatorCM.DeepCopy()
			updated.Data["OTHER_KEY"] = "new-value"
			result := handler.Predicates().UpdateFunc(event.UpdateEvent{
				ObjectOld: operatorCM,
				ObjectNew: updated,
			})
			Expect(result).To(BeFalse())
		})

		It("UpdateFunc returns false for wrong ConfigMap name", func() {
			wrongCM := operatorCM.DeepCopy()
			wrongCM.Name = "some-other-config"
			updated := wrongCM.DeepCopy()
			updated.Data[enableLimitedCacheKey] = "false"
			result := handler.Predicates().UpdateFunc(event.UpdateEvent{
				ObjectOld: wrongCM,
				ObjectNew: updated,
			})
			Expect(result).To(BeFalse())
		})

		It("UpdateFunc returns false for wrong namespace", func() {
			wrongNsCM := operatorCM.DeepCopy()
			wrongNsCM.Namespace = "default"
			updated := wrongNsCM.DeepCopy()
			updated.Data[enableLimitedCacheKey] = "false"
			result := handler.Predicates().UpdateFunc(event.UpdateEvent{
				ObjectOld: wrongNsCM,
				ObjectNew: updated,
			})
			Expect(result).To(BeFalse())
		})

		It("CreateFunc always returns false", func() {
			result := handler.Predicates().CreateFunc(event.CreateEvent{Object: operatorCM})
			Expect(result).To(BeFalse())
		})

		It("DeleteFunc always returns false", func() {
			result := handler.Predicates().DeleteFunc(event.DeleteEvent{Object: operatorCM})
			Expect(result).To(BeFalse())
		})
	})

	Describe("Reconcile", func() {
		sapBtpOperatorLabel := map[string]string{"app.kubernetes.io/instance": "sap-btp-operator"}

		newPod := func(name string, labels map[string]string) *corev1.Pod {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: operatorNamespace,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "mgr", Image: "fake"}},
				},
			}
		}

		It("deletes pods with the sap-btp-operator label and returns empty requests", func() {
			pod := newPod("btp-op-pod", sapBtpOperatorLabel)
			Expect(fakeClient.Create(context.Background(), pod)).To(Succeed())

			requests := handler.Reconcile(context.Background(), nil)

			Expect(requests).To(BeEmpty())
			err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(pod), &corev1.Pod{})
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("does not delete pods without the sap-btp-operator label", func() {
			pod := newPod("other-pod", map[string]string{"app": "something-else"})
			Expect(fakeClient.Create(context.Background(), pod)).To(Succeed())

			handler.Reconcile(context.Background(), nil)

			err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(pod), &corev1.Pod{})
			Expect(err).To(BeNil())
		})

		It("returns empty requests when no pods exist", func() {
			requests := handler.Reconcile(context.Background(), nil)
			Expect(requests).To(BeEmpty())
		})
	})
})
