package managerresource

import (
	"context"
	"testing"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/manifest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("returns resources to create when they are enabled", func() {
		manager := NewManager([]Resource{NewNetworkPolicies(true)}, &manifest.Handler{Scheme: scheme})
		resources, err := manager.ResourcesToCreate(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(HaveLen(4))
	})

	It("returns no resources to create when they are disabled", func() {
		manager := NewManager([]Resource{NewNetworkPolicies(false)}, &manifest.Handler{Scheme: scheme})
		resources, err := manager.ResourcesToCreate(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(BeEmpty())
	})

	It("returns resources to delete when they are disabled", func() {
		manager := NewManager([]Resource{NewNetworkPolicies(false)}, &manifest.Handler{Scheme: scheme})
		resources, err := manager.ResourcesToDelete(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(HaveLen(1))
	})

	It("returns no resources to delete when they are enabled", func() {
		manager := NewManager([]Resource{NewNetworkPolicies(true)}, &manifest.Handler{Scheme: scheme})
		resources, err := manager.ResourcesToDelete(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(BeEmpty())
	})
})
