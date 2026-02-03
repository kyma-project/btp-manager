package generic_test

import (
	"context"
	"testing"

	"github.com/kyma-project/btp-manager/internal/k8s/generic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	namespace     = "kyma-system"
	configmapName = "test-configmap"
)

var (
	fakeClient client.Client
	scheme     *runtime.Scheme
)

func TestObjectManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Object Manager Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
})

var _ = Describe("Object Manager", func() {
	var (
		configmapManager *generic.ObjectManager[*corev1.ConfigMap]
	)

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
	})

	Describe("for ConfigMaps", func() {

		BeforeEach(func() {
			configmapManager = generic.NewObjectManager[*corev1.ConfigMap](fakeClient)
		})

		Describe("Create ConfigMap", func() {
			Context("when the ConfigMap does not exist", func() {
				It("should create the ConfigMap", func() {
					configmap := configmap()

					Expect(configmapManager.Create(context.Background(), configmap)).To(Succeed())

					got := &corev1.ConfigMap{}
					Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(configmap), got)).To(Succeed())
					Expect(got).To(Equal(configmap))
				})
			})

			Context("when the ConfigMap already exists", func() {
				It("should return error while creating the ConfigMap", func() {
					existingConfigmap := configmap()
					configmap := existingConfigmap.DeepCopy()
					Expect(fakeClient.Create(context.Background(), existingConfigmap)).To(Succeed())

					err := configmapManager.Create(context.Background(), configmap)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("already exists"))
				})
			})
		})

		Describe("Apply ConfigMap", func() {
			const fieldOwner = "object-manager-test"

			Context("when the ConfigMap does not exist", func() {
				It("should apply the ConfigMap", func() {
					configmap := configmap()

					Expect(configmapManager.Apply(context.Background(), configmap, client.FieldOwner(fieldOwner))).To(Succeed())

					got := &corev1.ConfigMap{}
					Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(configmap), got)).To(Succeed())
					Expect(got).To(Equal(configmap))
				})
			})

			Context("when the ConfigMap already exists", func() {
				It("should apply the ConfigMap", func() {
					expectedData := map[string]string{"key2": "value2"}
					existingConfigmap, applyConfigmap := configmap(), configmap()
					applyConfigmap.Data = expectedData

					Expect(fakeClient.Create(context.Background(), existingConfigmap)).To(Succeed())

					Expect(configmapManager.Apply(context.Background(), applyConfigmap, client.FieldOwner(fieldOwner))).To(Succeed())

					got := &corev1.ConfigMap{}
					Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(existingConfigmap), got)).To(Succeed())
					Expect(got).To(Equal(applyConfigmap))
				})
			})
		})

		Describe("Get ConfigMap", func() {
			Context("when the ConfigMap exists", func() {
				It("should get the ConfigMap", func() {
					existingConfigmap := configmap()
					Expect(fakeClient.Create(context.Background(), existingConfigmap)).To(Succeed())

					got := &corev1.ConfigMap{}
					Expect(configmapManager.Get(context.Background(), client.ObjectKeyFromObject(existingConfigmap), got)).To(Succeed())
					Expect(got.Name).To(Equal(existingConfigmap.Name))
					Expect(got.Namespace).To(Equal(existingConfigmap.Namespace))
					Expect(got.Data).To(Equal(existingConfigmap.Data))
				})
			})

			Context("when the ConfigMap does not exist", func() {
				It("should return error while getting the ConfigMap", func() {
					configmap := configmap()

					got := &corev1.ConfigMap{}
					err := configmapManager.Get(context.Background(), client.ObjectKeyFromObject(configmap), got)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("not found"))
				})
			})
		})

		Describe("Update ConfigMap", func() {
			Context("when the ConfigMap exists", func() {
				It("should update the ConfigMap", func() {
					existingConfigmap := configmap()
					Expect(fakeClient.Create(context.Background(), existingConfigmap)).To(Succeed())

					updatedConfigmap := existingConfigmap.DeepCopy()
					updatedConfigmap.Data = map[string]string{"key1": "updatedValue"}

					Expect(configmapManager.Update(context.Background(), updatedConfigmap)).To(Succeed())

					got := &corev1.ConfigMap{}
					Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(existingConfigmap), got)).To(Succeed())
					Expect(got.Data).To(Equal(updatedConfigmap.Data))
				})
			})

			Context("when the ConfigMap does not exist", func() {
				It("should return error while updating the ConfigMap", func() {
					configmap := configmap()

					err := configmapManager.Update(context.Background(), configmap)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("not found"))
				})
			})
		})

		Describe("List ConfigMaps", func() {
			const (
				otherConfigmapName = "test-configmap-2"
				otherNamespace     = "test-namespace-2"
			)

			Context("when ConfigMaps exist", func() {
				It("should list all ConfigMaps", func() {
					configmap1 := configmap()
					configmap2 := configmap(otherConfigmapName, otherNamespace)
					Expect(fakeClient.Create(context.Background(), configmap1)).To(Succeed())
					Expect(fakeClient.Create(context.Background(), configmap2)).To(Succeed())

					list := &corev1.ConfigMapList{}
					Expect(configmapManager.List(context.Background(), list)).To(Succeed())

					Expect(list.Items).To(HaveLen(2))
					Expect(list.Items[0].Name).To(Or(Equal(configmap1.Name), Equal(configmap2.Name)))
					Expect(list.Items[1].Name).To(Or(Equal(configmap1.Name), Equal(configmap2.Name)))
				})

				It("should list ConfigMaps in a specific namespace", func() {
					configmapInNamespace := configmap()
					configmapInOtherNamespace := configmap(otherConfigmapName, otherNamespace)
					Expect(fakeClient.Create(context.Background(), configmapInNamespace)).To(Succeed())
					Expect(fakeClient.Create(context.Background(), configmapInOtherNamespace)).To(Succeed())

					list := &corev1.ConfigMapList{}
					Expect(configmapManager.List(context.Background(), list, client.InNamespace(namespace))).To(Succeed())

					Expect(list.Items).To(HaveLen(1))
					Expect(list.Items[0].Name).To(Equal(configmapInNamespace.Name))
					Expect(list.Items[0].Namespace).To(Equal(namespace))
				})

				It("should list ConfigMaps with matching labels", func() {
					expectedLabels := map[string]string{"app": "test"}
					configmapWithLabel := configmap()
					configmapWithLabel.Labels = expectedLabels
					configmapWithoutLabels := configmap(otherConfigmapName, otherNamespace)

					Expect(fakeClient.Create(context.Background(), configmapWithLabel)).To(Succeed())
					Expect(fakeClient.Create(context.Background(), configmapWithoutLabels)).To(Succeed())

					list := &corev1.ConfigMapList{}
					Expect(configmapManager.List(context.Background(), list, client.MatchingLabels(expectedLabels))).To(Succeed())

					Expect(list.Items).To(HaveLen(1))
					Expect(list.Items[0].Name).To(Equal(configmapWithLabel.Name))
					Expect(list.Items[0].Labels).To(Equal(configmapWithLabel.Labels))
				})
			})

			Context("when no ConfigMaps exist", func() {
				It("should return an empty list", func() {
					list := &corev1.ConfigMapList{}
					Expect(configmapManager.List(context.Background(), list)).To(Succeed())

					Expect(list.Items).To(BeEmpty())
				})
			})
		})

		Describe("Delete ConfigMap", func() {
			Context("when the ConfigMap exists", func() {
				It("should delete the ConfigMap", func() {
					existingConfigmap := configmap()
					Expect(fakeClient.Create(context.Background(), existingConfigmap)).To(Succeed())

					Expect(configmapManager.Delete(context.Background(), existingConfigmap)).To(Succeed())

					got := &corev1.ConfigMap{}
					err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(existingConfigmap), got)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("not found"))
				})
			})

			Context("when the ConfigMap does not exist", func() {
				It("should return error while deleting the ConfigMap", func() {
					configmap := configmap()

					err := configmapManager.Delete(context.Background(), configmap)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("not found"))
				})
			})
		})

	})
})

func configmap(nameAndNamespace ...string) *corev1.ConfigMap {
	name, namespace := configmapName, namespace
	if len(nameAndNamespace) > 0 {
		name = nameAndNamespace[0]
		namespace = nameAndNamespace[1]
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}
}
