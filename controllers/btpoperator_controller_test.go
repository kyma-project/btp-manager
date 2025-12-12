package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	statusUpdateTimeout       = time.Millisecond * 100
	statusUpdateCheckInterval = time.Millisecond * 20
)

func TestBtpOperatorReconciler_UpdateBtpOperatorStatus(t *testing.T) {
	ctx := context.Background()
	fakeK8sClient := fake.NewClientBuilder().Build()
	scheme := clientgoscheme.Scheme
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	btpOperator := createDefaultBtpOperator()
	require.NoError(t, fakeK8sClient.Create(ctx, btpOperator))
	config.StatusUpdateTimeout = statusUpdateTimeout
	config.StatusUpdateCheckInterval = statusUpdateCheckInterval

	t.Run("should return error from client.Get", func(t *testing.T) {
		// given
		retryK8sClient := newLazyK8sClient(fakeK8sClient, 3)
		btpOperatorReconciler := NewBtpOperatorReconciler(retryK8sClient, fakeK8sClient, scheme, nil, nil)
		retryK8sClient.EnableErrorOnGet()

		// when
		err := btpOperatorReconciler.UpdateBtpOperatorStatus(ctx, btpOperator, v1alpha1.StateProcessing, conditions.Initialized, "test")

		// then
		require.Error(t, err)
		assert.Equal(t, k8sClientGetPermanentErrMsg, err.Error())

		// when
		currentBtpOperator := &v1alpha1.BtpOperator{}
		err = fakeK8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), currentBtpOperator)

		// then
		require.NoError(t, err)
		assert.Equal(t, "", string(currentBtpOperator.Status.State))
		assert.Equal(t, 0, len(currentBtpOperator.Status.Conditions))
	})

	t.Run("should return error from client.Update", func(t *testing.T) {
		// given
		retryK8sClient := newLazyK8sClient(fakeK8sClient, 3)
		btpOperatorReconciler := NewBtpOperatorReconciler(retryK8sClient, fakeK8sClient, scheme, nil, nil)
		retryK8sClient.EnableErrorOnUpdate()

		// when
		err := btpOperatorReconciler.UpdateBtpOperatorStatus(ctx, btpOperator, v1alpha1.StateProcessing, conditions.Initialized, "test")

		// then
		require.Error(t, err)
		assert.Equal(t, k8sClientUpdatePermanentErrMsg, err.Error())

		// when
		currentBtpOperator := &v1alpha1.BtpOperator{}
		err = fakeK8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), currentBtpOperator)

		// then
		require.NoError(t, err)
		assert.Equal(t, "", string(currentBtpOperator.Status.State))
		assert.Equal(t, 0, len(currentBtpOperator.Status.Conditions))
	})

	t.Run("should time out", func(t *testing.T) {
		// given
		disabledUpdatek8sClient := newLazyK8sClient(fakeK8sClient, 3)
		btpOperatorReconciler := NewBtpOperatorReconciler(disabledUpdatek8sClient, fakeK8sClient, scheme, nil, nil)
		disabledUpdatek8sClient.DisableUpdate()

		// when
		err := btpOperatorReconciler.UpdateBtpOperatorStatus(ctx, btpOperator, v1alpha1.StateProcessing, conditions.Initialized, "test")

		// then
		require.NoError(t, err)

		// when
		currentBtpOperator := &v1alpha1.BtpOperator{}
		err = fakeK8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), currentBtpOperator)

		// then
		require.NoError(t, err)
		assert.Equal(t, "", string(currentBtpOperator.Status.State))
		assert.Equal(t, 0, len(currentBtpOperator.Status.Conditions))
	})

	t.Run("should update BtpOperator status after a few retries", func(t *testing.T) {
		// given
		retryK8sClient := newLazyK8sClient(fakeK8sClient, 3)
		btpOperatorReconciler := NewBtpOperatorReconciler(retryK8sClient, fakeK8sClient, scheme, nil, nil)

		// when
		err := btpOperatorReconciler.UpdateBtpOperatorStatus(ctx, btpOperator, v1alpha1.StateProcessing, conditions.Initialized, "test")

		// then
		require.NoError(t, err)

		// when
		currentBtpOperator := &v1alpha1.BtpOperator{}
		err = fakeK8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), currentBtpOperator)

		// then
		require.NoError(t, err)
		assert.Equal(t, string(v1alpha1.StateProcessing), string(currentBtpOperator.Status.State))
		assert.Equal(t, 1, len(currentBtpOperator.Status.Conditions))
		assert.True(t, currentBtpOperator.IsReasonStringEqual(string(conditions.Initialized)))
	})

	t.Run("should update BtpOperator status three times", func(t *testing.T) {
		// given
		retryK8sClient := newLazyK8sClient(fakeK8sClient, 3)
		btpOperatorReconciler := NewBtpOperatorReconciler(retryK8sClient, fakeK8sClient, scheme, nil, nil)
		conditionMsg1 := "test1"
		conditionMsg2 := "test2"
		conditionMsg3 := "test3"

		// when
		err := btpOperatorReconciler.UpdateBtpOperatorStatus(ctx, btpOperator, v1alpha1.StateProcessing, conditions.Initialized, conditionMsg1)

		// then
		require.NoError(t, err)

		// when
		currentBtpOperator := &v1alpha1.BtpOperator{}
		err = fakeK8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), currentBtpOperator)

		// then
		require.NoError(t, err)
		assert.Equal(t, string(v1alpha1.StateProcessing), string(currentBtpOperator.Status.State))
		assert.Equal(t, 1, len(currentBtpOperator.Status.Conditions))
		assert.True(t, currentBtpOperator.IsMsgForGivenReasonEqual(string(conditions.Initialized), conditionMsg1))

		// when
		err = btpOperatorReconciler.UpdateBtpOperatorStatus(ctx, btpOperator, v1alpha1.StateReady, conditions.ReconcileSucceeded, conditionMsg2)

		// then
		require.NoError(t, err)

		// when
		currentBtpOperator = &v1alpha1.BtpOperator{}
		err = fakeK8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), currentBtpOperator)

		// then
		require.NoError(t, err)
		assert.Equal(t, string(v1alpha1.StateReady), string(currentBtpOperator.Status.State))
		assert.Equal(t, 1, len(currentBtpOperator.Status.Conditions))
		assert.True(t, currentBtpOperator.IsMsgForGivenReasonEqual(string(conditions.ReconcileSucceeded), conditionMsg2))

		// when
		err = btpOperatorReconciler.UpdateBtpOperatorStatus(ctx, btpOperator, v1alpha1.StateReady, conditions.ReconcileSucceeded, conditionMsg3)

		// then
		require.NoError(t, err)

		// when
		currentBtpOperator = &v1alpha1.BtpOperator{}
		err = fakeK8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), currentBtpOperator)

		// then
		require.NoError(t, err)
		assert.Equal(t, string(v1alpha1.StateReady), string(currentBtpOperator.Status.State))
		assert.Equal(t, 1, len(currentBtpOperator.Status.Conditions))
		assert.True(t, currentBtpOperator.IsMsgForGivenReasonEqual(string(conditions.ReconcileSucceeded), conditionMsg3))
	})
}
