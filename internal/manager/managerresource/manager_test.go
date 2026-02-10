package managerresource

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/manager/moduleresource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	ManagerResourcesPath = "./testdata"
)

var (
	fakeClient client.Client
	scheme     *runtime.Scheme
)

func TestManagerResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Manager Resource Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	config.ManagerResourcesPath = ManagerResourcesPath
})

var _ = Describe("Resource Manager", func() {
	var (
		ctx     context.Context
		manager *Manager
	)

	BeforeEach(func() {
		ctx = context.Background()
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		manager = NewManager(fakeClient, scheme, []Resource{&NetworkPolicies{}})
	})

	It("should create enabled manager resources", func() {
		Expect(createBtpOperatorCR(ctx, fakeClient, "false")).To(Succeed())

		err := manager.createManagerResources(ctx)
		Expect(err).NotTo(HaveOccurred())

		expectNetworkPoliciesCount(ctx, fakeClient, 4)
	})

	It("should skip disabled manager resources", func() {
		Expect(createBtpOperatorCR(ctx, fakeClient, "true")).To(Succeed())

		err := manager.createManagerResources(ctx)
		Expect(err).NotTo(HaveOccurred())

		expectNetworkPoliciesCount(ctx, fakeClient, 0)
	})

	It("should delete disabled manager resources", func() {
		Expect(createBtpOperatorCR(ctx, fakeClient, "true")).To(Succeed())
		Expect(createNetworkPolicies(ctx, fakeClient, 4)).To(Succeed())

		err := manager.deleteManagerResources(ctx)
		Expect(err).NotTo(HaveOccurred())

		expectNetworkPoliciesCount(ctx, fakeClient, 0)
	})

	It("should not delete enabled manager resources", func() {
		Expect(createBtpOperatorCR(ctx, fakeClient, "false")).To(Succeed())
		Expect(createNetworkPolicies(ctx, fakeClient, 4)).To(Succeed())

		err := manager.deleteManagerResources(ctx)
		Expect(err).NotTo(HaveOccurred())

		expectNetworkPoliciesCount(ctx, fakeClient, 4)
	})
})

func expectNetworkPoliciesCount(ctx context.Context, k8sClient client.Client, expected int) {
	var policies networkingv1.NetworkPolicyList
	err := k8sClient.List(ctx, &policies, client.InNamespace(config.KymaSystemNamespaceName))
	Expect(err).NotTo(HaveOccurred())
	Expect(policies.Items).To(HaveLen(expected))
}

func createBtpOperatorCR(ctx context.Context, k8sClient client.Client, disableNetworkPolicies string) error {
	return k8sClient.Create(ctx, &v1alpha1.BtpOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.BtpOperatorCrName,
			Namespace: config.KymaSystemNamespaceName,
			Annotations: map[string]string{
				v1alpha1.DisableNetworkPoliciesAnnotation: disableNetworkPolicies,
			},
		},
	})
}

func createNetworkPolicies(ctx context.Context, k8sClient client.Client, count int) error {
	for i := 1; i <= count; i++ {
		policy := &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-policy-%d", i),
				Namespace: config.KymaSystemNamespaceName,
				Labels: map[string]string{
					moduleresource.ManagedByLabelKey: moduleresource.OperatorName,
				},
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
			},
		}
		if err := k8sClient.Create(ctx, policy); err != nil {
			return err
		}
	}
	return nil
}
