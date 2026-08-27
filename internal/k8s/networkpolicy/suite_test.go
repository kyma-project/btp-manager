package networkpolicy_test

import (
	"testing"

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
	kymaNamespace = "kyma-system"

	policyName1 = "kyma-project.io--btp-operator-allow-to-apiserver"
	policyName2 = "kyma-project.io--btp-operator-to-dns"

	managedByLabelKey         = "app.kubernetes.io/managed-by"
	kymaProjectModuleLabelKey = "kyma-project.io/module"
	operatorName              = "btp-manager"
	moduleName                = "btp-operator"
)

var (
	fakeClient client.Client
	scheme     *runtime.Scheme
)

func TestNetworkPolicyManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network Policy Manager Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
})

func managedNetworkPolicy(name string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kymaNamespace,
			Labels: map[string]string{
				managedByLabelKey:         operatorName,
				kymaProjectModuleLabelKey: moduleName,
			},
		},
	}
}

func unmanagedNetworkPolicy(name string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kymaNamespace,
		},
	}
}

func newFakeClient(objects ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()
}
