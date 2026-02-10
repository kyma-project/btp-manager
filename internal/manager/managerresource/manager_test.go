package managerresource

import (
	"context"
	"testing"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/manifest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	ManagerResourcesPath = "./testdata"
)

var (
	scheme *runtime.Scheme
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
		manager = NewManager([]Resource{&NetworkPolicies{}}, &manifest.Handler{Scheme: scheme})
	})

	It("returns resources when they are enabled", func() {
		resources, err := manager.ResourcesToCreate(ctx, createBtpOperatorCR("false"))
		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(HaveLen(4))
	})

	It("returns no resources to create when they are disabled", func() {
		resources, err := manager.ResourcesToCreate(ctx, createBtpOperatorCR("true"))
		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(BeEmpty())
	})

	It("returns resources to delete when they are disabled", func() {
		resources, err := manager.ResourcesToDelete(ctx, createBtpOperatorCR("true"))
		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(HaveLen(1))
	})

	It("returns no resources to delete when they are enabled", func() {
		resources, err := manager.ResourcesToDelete(ctx, createBtpOperatorCR("false"))
		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(BeEmpty())
	})
})

func createBtpOperatorCR(disableNetworkPolicies string) *v1alpha1.BtpOperator {
	return &v1alpha1.BtpOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.BtpOperatorCrName,
			Namespace: config.KymaSystemNamespaceName,
			Annotations: map[string]string{
				v1alpha1.DisableNetworkPoliciesAnnotation: disableNetworkPolicies,
			},
		},
	}
}
