package deprovisioning

import (
	"testing"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsForceDelete_LabelTrue(t *testing.T) {
	cr := &v1alpha1.BtpOperator{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{forceDeleteLabelKey: "true"}}}
	if !IsForceDelete(cr) {
		t.Fatal("expected force delete = true")
	}
}

func TestIsForceDelete_LabelAbsent(t *testing.T) {
	cr := &v1alpha1.BtpOperator{}
	if IsForceDelete(cr) {
		t.Fatal("expected force delete = false")
	}
}

func TestGvkToList(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "g", Version: "v1", Kind: "Foo"}
	list := GvkToList(gvk)
	if list.GetKind() != "FooList" {
		t.Fatalf("expected FooList, got %s", list.GetKind())
	}
}
