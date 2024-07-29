package servicemanager_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	// given
	secretProvider := clusterobject.NewFakeSecretManager()
	secretProvider.Create(clusterobject.FakeDefaultSecret())
	srv, err := servicemanager.NewFakeServer()
	require.NoError(t, err)

	srv.Start()
	defer srv.Close()
	httpClient := srv.Client()
	url := srv.URL

	defaultServiceOfferings, err := servicemanager.GetServiceOfferingsFromJSON()
	require.NoError(t, err)
	defaultServicePlans, err := servicemanager.GetServicePlansFromJSON()
	require.NoError(t, err)
	defaultServiceInstances, err := servicemanager.GetServiceInstancesFromJSON()
	require.NoError(t, err)
	defaultServiceBindings, err := servicemanager.GetServiceBindingsFromJSON()
	require.NoError(t, err)

	t.Run("should get service offerings available for the default credentials", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)

		// when
		err = smClient.Defaults(ctx)

		// then
		require.NoError(t, err)

		// given
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)

		// when
		sos, err := smClient.ServiceOfferings()

		// then
		require.NoError(t, err)
		assertEqualServiceOfferings(t, defaultServiceOfferings, sos)
	})

	t.Run("should get service offering details and plans for given service offering ID", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		soID := "fc26622b-aeb2-4f3c-95da-8eb337a26883"
		expectedServiceOffering := getServiceOfferingByID(defaultServiceOfferings, soID)
		filteredServicePlans := filterServicePlansByServiceOfferingID(defaultServicePlans, soID)

		// when
		sod, err := smClient.ServiceOfferingDetails(soID)

		// then
		require.NoError(t, err)
		assert.Len(t, sod.ServicePlans.Items, 3)
		assertEqualServiceOffering(t, *expectedServiceOffering, sod.ServiceOffering)
		assertEqualServicePlans(t, &filteredServicePlans, &sod.ServicePlans)
	})

	t.Run("should get all service instances", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)

		// when
		sis, err := smClient.ServiceInstances()

		// then
		require.NoError(t, err)
		assertEqualServiceInstances(t, defaultServiceInstances, sis)
	})

	t.Run("should get service instance for given service instance ID", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "c7a604e8-f289-4f61-841f-c6519db8daf2"
		expectedServiceInstance := getServiceInstanceByID(defaultServiceInstances, siID)

		// when
		si, err := smClient.ServiceInstance(siID)

		// then
		require.NoError(t, err)
		assertEqualServiceInstance(t, *expectedServiceInstance, *si)
	})

	t.Run("should create service instance", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siCreateRequest := &types.ServiceInstance{
			Common: types.Common{
				Name: "test-service-instance",
				Labels: types.Labels{
					"test-label":         []string{"test-value"},
					types.K8sNameLabel:   []string{"test-service-instance"},
					types.NamespaceLabel: []string{"test-namespace"},
					types.ClusterIDLabel: []string{"test-cluster-id"},
				},
			},
			ServicePlanID: "test-service-plan-id",
			Parameters:    json.RawMessage(`{"test-parameter": "test-value"}`),
		}

		// when
		si, err := smClient.CreateServiceInstance(siCreateRequest)

		// then
		require.NoError(t, err)
		assert.NotEmpty(t, si.ID)
		assert.Equal(t, siCreateRequest.Name, si.Name)
		assert.Equal(t, siCreateRequest.ServicePlanID, si.ServicePlanID)
		assert.Equal(t, siCreateRequest.Labels, si.Labels)

		var expectedParams, actualParams []byte
		require.NoError(t, siCreateRequest.Parameters.UnmarshalJSON(expectedParams))
		require.NoError(t, si.Parameters.UnmarshalJSON(actualParams))
		assert.Equal(t, expectedParams, actualParams)
	})

	t.Run("should update service instance except shared field", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "f9ffbaa4-739a-4a16-ad02-6f2f17a830c5"
		siUpdatedName := "updated-service-instance"
		siUpdatedServicePlanID := "updated-service-plan-id"
		siUpdatedParameters := json.RawMessage(`{"updated-parameter": "updated-value"}`)
		siUpdatedLabels := []types.LabelChange{{Operation: types.AddLabelOperation, Key: "updated-label", Values: []string{"updated-value"}}}
		siUpdateRequest := &types.ServiceInstanceUpdateRequest{
			ID:            &siID,
			Name:          &siUpdatedName,
			ServicePlanID: &siUpdatedServicePlanID,
			Parameters:    &siUpdatedParameters,
			Labels:        siUpdatedLabels,
		}

		// when
		si, err := smClient.UpdateServiceInstance(siUpdateRequest)

		// then
		require.NoError(t, err)
		assert.Equal(t, siID, si.ID)
		assert.Equal(t, siUpdatedName, si.Name)
		assert.Equal(t, siUpdatedServicePlanID, si.ServicePlanID)
		assert.Contains(t, si.Labels, siUpdatedLabels[0].Key)

		var expectedParams, actualParams []byte
		require.NoError(t, siUpdatedParameters.UnmarshalJSON(expectedParams))
		require.NoError(t, si.Parameters.UnmarshalJSON(actualParams))
		assert.Equal(t, expectedParams, actualParams)

		// revert server to default state
		srv, err = servicemanager.NewFakeServer()
		require.NoError(t, err)
	})

	t.Run("should update only the shared field in a service instance", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "df28885c-7c5f-46f0-bb75-0ae2dc85ac41"
		siUpdatedShared := true
		siUpdateRequest := &types.ServiceInstanceUpdateRequest{
			ID:     &siID,
			Shared: &siUpdatedShared,
		}

		// when
		si, err := smClient.UpdateServiceInstance(siUpdateRequest)

		// then
		require.NoError(t, err)
		assert.Equal(t, siID, si.ID)
		assert.Equal(t, siUpdatedShared, si.Shared)

		// revert server to default state
		srv, err = servicemanager.NewFakeServer()
		require.NoError(t, err)
	})

	t.Run("delete service instance", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "a7e240d6-e348-4fc0-a54c-7b7bfe9b9da6"

		// when
		err := smClient.DeleteServiceInstance(siID)

		// then
		require.NoError(t, err)

		// when
		_, err = smClient.ServiceInstance(siID)

		// then
		require.Error(t, err)

		// revert server to default state
		srv, err = servicemanager.NewFakeServer()
		require.NoError(t, err)
	})

	t.Run("should get all service bindings", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)

		// when
		sbs, err := smClient.ServiceBindings()

		// then
		require.NoError(t, err)
		assertEqualServiceBindings(t, defaultServiceBindings, sbs)
	})

	t.Run("should get service binding for given service binding ID", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		sbID := "550e8400-e29b-41d4-a716-446655440003"
		expectedServiceBinding := getServiceBindingByID(defaultServiceBindings, sbID)

		// when
		sb, err := smClient.ServiceBinding(sbID)

		// then
		require.NoError(t, err)
		assertEqualServiceBinding(t, *expectedServiceBinding, *sb)
	})

	t.Run("should create service binding", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		sbCreateRequest := &types.ServiceBinding{
			Common: types.Common{
				Name:   "test-service-binding",
				Labels: types.Labels{"test-label": []string{"test-value"}},
			},
			ServiceInstanceID: "test-service-instance-id",
			Parameters:        json.RawMessage(`{"test-parameter": "test-value"}`),
			BindResource:      json.RawMessage(`{"test-bind-resource": "test-value"}`),
		}

		// when
		sb, err := smClient.CreateServiceBinding(sbCreateRequest)

		// then
		require.NoError(t, err)
		assert.NotEmpty(t, sb.ID)
		assert.Equal(t, sbCreateRequest.Name, sb.Name)
		assert.Equal(t, sbCreateRequest.ServiceInstanceID, sb.ServiceInstanceID)
		assert.Equal(t, sbCreateRequest.Labels, sb.Labels)

		var expectedParams, actualParams []byte
		require.NoError(t, sbCreateRequest.Parameters.UnmarshalJSON(expectedParams))
		require.NoError(t, sb.Parameters.UnmarshalJSON(actualParams))
		assert.Equal(t, expectedParams, actualParams)

		var expectedBindResource, actualBindResource []byte
		require.NoError(t, sbCreateRequest.Parameters.UnmarshalJSON(expectedBindResource))
		require.NoError(t, sb.Parameters.UnmarshalJSON(actualBindResource))
		assert.Equal(t, expectedBindResource, actualBindResource)
	})

	t.Run("should delete service binding", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		sbID := "318a16c3-7c80-485f-b55c-918629012c9a"

		// when
		err := smClient.DeleteServiceBinding(sbID)

		// then
		require.NoError(t, err)

		// when
		_, err = smClient.ServiceBinding(sbID)

		// then
		require.Error(t, err)

		// revert server to default state
		srv, err = servicemanager.NewFakeServer()
		require.NoError(t, err)
	})
}

func getServiceOfferingByID(serviceOfferings *types.ServiceOfferings, serviceOfferingID string) *types.ServiceOffering {
	for _, so := range serviceOfferings.Items {
		if so.ID == serviceOfferingID {
			return &so
		}
	}
	return nil
}

func getServiceInstanceByID(serviceInstances *types.ServiceInstances, serviceInstanceID string) *types.ServiceInstance {
	for _, si := range serviceInstances.Items {
		if si.ID == serviceInstanceID {
			return &si
		}
	}
	return nil
}

func getServiceBindingByID(serviceBindings *types.ServiceBindings, serviceBindingID string) *types.ServiceBinding {
	for _, sb := range serviceBindings.Items {
		if sb.ID == serviceBindingID {
			return &sb
		}
	}
	return nil
}

func filterServicePlansByServiceOfferingID(servicePlans *types.ServicePlans, serviceOfferingID string) types.ServicePlans {
	var filteredSp types.ServicePlans
	for _, sp := range servicePlans.Items {
		if sp.ServiceOfferingID == serviceOfferingID {
			filteredSp.Items = append(filteredSp.Items, sp)
		}
	}
	return filteredSp
}

func assertEqualServiceOfferings(t *testing.T, expected, actual *types.ServiceOfferings) {
	assert.Len(t, actual.Items, len(expected.Items))
	for i := 0; i < len(expected.Items); i++ {
		expectedToCompare, actualToCompare := expected.Items[i], actual.Items[i]
		assertEqualServiceOffering(t, expectedToCompare, actualToCompare)
	}
}

func assertEqualServiceOffering(t *testing.T, expectedToCompare types.ServiceOffering, actualToCompare types.ServiceOffering) {
	var expectedBuff, actualBuff []byte
	require.NoError(t, expectedToCompare.Metadata.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Metadata.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Metadata, actualToCompare.Metadata = nil, nil

	require.NoError(t, expectedToCompare.Tags.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Tags.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Tags, actualToCompare.Tags = nil, nil

	assert.Equal(t, expectedToCompare, actualToCompare)
}

func assertEqualServicePlans(t *testing.T, expected, actual *types.ServicePlans) {
	assert.Len(t, actual.Items, len(expected.Items))
	for i := 0; i < len(expected.Items); i++ {
		expectedToCompare, actualToCompare := expected.Items[i], actual.Items[i]
		assertEqualServicePlan(t, expectedToCompare, actualToCompare)
	}
}

func assertEqualServicePlan(t *testing.T, expectedToCompare types.ServicePlan, actualToCompare types.ServicePlan) {
	var expectedBuff, actualBuff []byte
	require.NoError(t, expectedToCompare.Metadata.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Metadata.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Metadata, actualToCompare.Metadata = nil, nil

	require.NoError(t, expectedToCompare.Schemas.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Schemas.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Schemas, actualToCompare.Schemas = nil, nil

	assert.Equal(t, expectedToCompare, actualToCompare)
}

func assertEqualServiceInstances(t *testing.T, expected, actual *types.ServiceInstances) {
	assert.Len(t, actual.Items, len(expected.Items))
	for i := 0; i < len(expected.Items); i++ {
		expectedToCompare, actualToCompare := expected.Items[i], actual.Items[i]
		assertEqualServiceInstance(t, expectedToCompare, actualToCompare)
	}
}

func assertEqualServiceInstance(t *testing.T, expectedToCompare types.ServiceInstance, actualToCompare types.ServiceInstance) {
	var expectedBuff, actualBuff []byte
	require.NoError(t, expectedToCompare.Parameters.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Parameters.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Parameters, actualToCompare.Parameters = nil, nil

	require.NoError(t, expectedToCompare.MaintenanceInfo.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.MaintenanceInfo.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.MaintenanceInfo, actualToCompare.MaintenanceInfo = nil, nil

	require.NoError(t, expectedToCompare.Context.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Context.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Context, actualToCompare.Context = nil, nil

	require.NoError(t, expectedToCompare.PreviousValues.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.PreviousValues.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.PreviousValues, actualToCompare.PreviousValues = nil, nil

	assert.Equal(t, expectedToCompare, actualToCompare)
}

func assertEqualServiceBindings(t *testing.T, expected, actual *types.ServiceBindings) {
	assert.Len(t, actual.Items, len(expected.Items))
	for i := 0; i < len(expected.Items); i++ {
		expectedToCompare, actualToCompare := expected.Items[i], actual.Items[i]
		assertEqualServiceBinding(t, expectedToCompare, actualToCompare)
	}
}

func assertEqualServiceBinding(t *testing.T, expectedToCompare, actualToCompare types.ServiceBinding) {
	var expectedBuff, actualBuff []byte
	require.NoError(t, expectedToCompare.Credentials.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Credentials.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Credentials, actualToCompare.Credentials = nil, nil

	require.NoError(t, expectedToCompare.VolumeMounts.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.VolumeMounts.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.VolumeMounts, actualToCompare.VolumeMounts = nil, nil

	require.NoError(t, expectedToCompare.Endpoints.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Endpoints.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Endpoints, actualToCompare.Endpoints = nil, nil

	require.NoError(t, expectedToCompare.Context.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Context.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Context, actualToCompare.Context = nil, nil

	require.NoError(t, expectedToCompare.Parameters.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Parameters.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Parameters, actualToCompare.Parameters = nil, nil

	require.NoError(t, expectedToCompare.BindResource.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.BindResource.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.BindResource, actualToCompare.BindResource = nil, nil

	assert.Equal(t, expectedToCompare, actualToCompare)
}
