package autoarchivesettings

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	automated_archival "github.com/signalfx/signalfx-go/automated-archival"
)

// MockClient mocks the SignalFx client for testing
type MockClient struct {
	mock.Mock
}

func (m *MockClient) GetSettings(ctx context.Context) (*automated_archival.AutomatedArchivalSettings, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*automated_archival.AutomatedArchivalSettings), args.Error(1)
}

func (m *MockClient) CreateSettings(ctx context.Context, settings *automated_archival.AutomatedArchivalSettings) (*automated_archival.AutomatedArchivalSettings, error) {
	args := m.Called(ctx, settings)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*automated_archival.AutomatedArchivalSettings), args.Error(1)
}

func (m *MockClient) UpdateSettings(ctx context.Context, settings *automated_archival.AutomatedArchivalSettings) (*automated_archival.AutomatedArchivalSettings, error) {
	args := m.Called(ctx, settings)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*automated_archival.AutomatedArchivalSettings), args.Error(1)
}

func (m *MockClient) DeleteSettings(ctx context.Context, request *automated_archival.AutomatedArchivalSettingsDeleteRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

// Helper function to create test provider meta with mock client
func testProviderMeta(client *MockClient) map[string]interface{} {
	return map[string]interface{}{
		"client": client,
	}
}

func TestResourceRead(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	// Mock settings that will be returned by the API
	mockSettings := &automated_archival.AutomatedArchivalSettings{
		Version:        123,
		Enabled:        true,
		LookbackPeriod: "P30D",
		GracePeriod:    "P30D",
		RulesetLimit:   int32Ptr(100),
	}

	mockClient.On("GetSettings", ctx).Return(mockSettings, nil)

	d := schema.TestResourceDataRaw(t, newSchema(), nil)
	d.SetId("123")

	// Test the read function
	diags := resourceRead(ctx, d, testProviderMeta(mockClient))

	// Verify results
	assert.False(t, diags.HasError())
	mockClient.AssertExpectations(t)
}

func int32Ptr(i int32) *int32 {
	return &i
}

func TestResourceRead_Error(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	mockError := errors.New("API error")
	mockClient.On("GetSettings", ctx).Return(nil, mockError)

	d := schema.TestResourceDataRaw(t, newSchema(), nil)
	d.SetId("123")

	diags := resourceRead(ctx, d, testProviderMeta(mockClient))

	assert.True(t, diags.HasError())
	mockClient.AssertExpectations(t)
}

func TestResourceCreate(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	// Mock response from create API
	createdSettings := &automated_archival.AutomatedArchivalSettings{
		Version:        123,
		Enabled:        true,
		LookbackPeriod: "P30D",
		GracePeriod:    "P30D",
		RulesetLimit:   int32Ptr(100),
	}

	mockClient.On("CreateSettings", ctx, mock.Anything).Return(createdSettings, nil)

	d := schema.TestResourceDataRaw(t, newSchema(), map[string]interface{}{
		"enabled":         true,
		"lookback_period": "P30D",
		"grace_period":    "P30D",
		"ruleset_limit":   int32Ptr(100),
	})

	diags := resourceCreate(ctx, d, testProviderMeta(mockClient))

	assert.False(t, diags.HasError())
	assert.Equal(t, "123", d.Id())
	mockClient.AssertExpectations(t)
}

func TestResourceUpdate(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	updatedSettings := &automated_archival.AutomatedArchivalSettings{
		Version:        123,
		Enabled:        false,
		LookbackPeriod: "P30D",
		GracePeriod:    "P30D",
		RulesetLimit:   int32Ptr(200),
	}

	mockClient.On("UpdateSettings", ctx, mock.Anything).Return(updatedSettings, nil)

	d := schema.TestResourceDataRaw(t, newSchema(), map[string]interface{}{
		"enabled":         false,
		"lookback_period": "P30D",
		"grace_period":    "P30D",
		"ruleset_limit":   int32Ptr(200),
	})
	d.SetId("123")

	diags := resourceUpdate(ctx, d, testProviderMeta(mockClient))

	assert.False(t, diags.HasError())
	mockClient.AssertExpectations(t)
}

func TestResourceDelete(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	version := int64(123)
	mockClient.On("DeleteSettings", ctx, &automated_archival.AutomatedArchivalSettingsDeleteRequest{
		Version: &version,
	}).Return(nil)

	d := schema.TestResourceDataRaw(t, newSchema(), nil)
	d.SetId("123")

	diags := resourceDelete(ctx, d, testProviderMeta(mockClient))

	assert.False(t, diags.HasError())
	assert.Equal(t, "", d.Id())
	mockClient.AssertExpectations(t)
}

func TestResourceDelete_Error(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	mockError := errors.New("delete error")
	mockClient.On("DeleteSettings", ctx, mock.Anything).Return(mockError)

	d := schema.TestResourceDataRaw(t, newSchema(), nil)
	d.SetId("123")

	diags := resourceDelete(ctx, d, testProviderMeta(mockClient))

	assert.True(t, diags.HasError())
	mockClient.AssertExpectations(t)
}
