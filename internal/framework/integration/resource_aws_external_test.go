// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceAWSExternalMetadata(t *testing.T) {
	t.Parallel()

	implementation := NewResourceAWSExternal()
	response := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, response)

	assert.Equal(t, "signalfx_aws_external_integration", response.TypeName)
}

func TestResourceAWSExternalSchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourceAWSExternal(), resourceAWSExternalModel{}))

	response := &resource.SchemaResponse{}
	NewResourceAWSExternal().Schema(context.Background(), resource.SchemaRequest{}, response)
	name, ok := response.Schema.Attributes["name"].(schema.StringAttribute)
	require.True(t, ok)
	assert.Len(t, name.PlanModifiers, 1)
	for _, attributeName := range []string{"external_id", "signalfx_aws_account"} {
		attribute, attributeOK := response.Schema.Attributes[attributeName].(schema.StringAttribute)
		require.True(t, attributeOK)
		assert.True(t, attribute.Computed)
		assert.True(t, attribute.Sensitive)
	}
}

func TestResourceAWSExternalModel(t *testing.T) {
	t.Parallel()

	model := resourceAWSExternalModel{
		awsBootstrapModel: awsBootstrapModel{
			ID:                 types.StringValue("aws-id"),
			Name:               types.StringValue("Primary AWS external"),
			SignalFxAWSAccount: types.StringValue("arn:aws:iam::111111111111:root"),
		},
		ExternalID: types.StringValue("external-value"),
	}

	assert.Equal(t, &integration.AwsCloudWatchIntegration{
		Type:       "AWSCloudWatch",
		AuthMethod: integration.EXTERNAL_ID,
		Name:       "Primary AWS external",
		PollRate:   300000,
	}, model.awsIntegration(integration.EXTERNAL_ID))

	model.updateFromAPI(nil, true)
	assert.Equal(t, types.StringValue("aws-id"), model.ID)

	model.updateFromAPI(&integration.AwsCloudWatchIntegration{
		Id: "updated-id", Name: "Updated by API", AuthMethod: integration.EXTERNAL_ID,
		ExternalId: "updated-external", SfxAwsAccountArn: "arn:aws:iam::222222222222:root",
	}, true)
	assert.Equal(t, types.StringValue("updated-id"), model.ID)
	assert.Equal(t, types.StringValue("Updated by API"), model.Name)
	assert.Equal(t, types.StringValue("updated-external"), model.ExternalID)
	assert.Equal(t, types.StringValue("arn:aws:iam::222222222222:root"), model.SignalFxAWSAccount)

	model.updateFromAPI(&integration.AwsCloudWatchIntegration{
		Id: "ignored-id", Name: "Ignored read name", AuthMethod: integration.SECURITY_TOKEN,
		SfxAwsAccountArn: "arn:aws:iam::333333333333:root",
	}, false)
	assert.Equal(t, types.StringValue("updated-id"), model.ID)
	assert.Equal(t, types.StringValue("Updated by API"), model.Name)
	assert.Equal(t, types.StringValue("updated-external"), model.ExternalID, "non-external reads must retain the generated external ID")
	assert.Equal(t, types.StringValue("arn:aws:iam::333333333333:root"), model.SignalFxAWSAccount)
}

func TestResourceAWSExternalRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	implementation := &ResourceAWSExternal{}
	schemaResponse := &resource.SchemaResponse{}
	implementation.Schema(ctx, resource.SchemaRequest{}, schemaResponse)
	invalidPlan := tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Bool, true), Schema: schemaResponse.Schema}
	invalidState := tfsdk.State{Raw: tftypes.NewValue(tftypes.Bool, true), Schema: schemaResponse.Schema}

	createResponse := &resource.CreateResponse{}
	implementation.Create(ctx, resource.CreateRequest{Plan: invalidPlan}, createResponse)
	assert.True(t, createResponse.Diagnostics.HasError())
	readResponse := &resource.ReadResponse{}
	implementation.Read(ctx, resource.ReadRequest{State: invalidState}, readResponse)
	assert.True(t, readResponse.Diagnostics.HasError())
	updateResponse := &resource.UpdateResponse{}
	implementation.Update(ctx, resource.UpdateRequest{Plan: invalidPlan, State: invalidState}, updateResponse)
	assert.True(t, updateResponse.Diagnostics.HasError())
	deleteResponse := &resource.DeleteResponse{}
	implementation.Delete(ctx, resource.DeleteRequest{State: invalidState}, deleteResponse)
	assert.True(t, deleteResponse.Diagnostics.HasError())
}

func TestResourceAWSExternalMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	createdNames := make([]string, 0, 2)
	stored := map[string]integration.AwsCloudWatchIntegration{}

	writeStored := func(w http.ResponseWriter, id string) {
		mu.Lock()
		current := stored[id]
		mu.Unlock()
		if err := json.NewEncoder(w).Encode(current); err != nil {
			t.Errorf("write AWS external response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload integration.AwsCloudWatchIntegration
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode AWS external payload: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assert.Equal(t, integration.Type("AWSCloudWatch"), payload.Type)
			assert.Equal(t, integration.EXTERNAL_ID, payload.AuthMethod)
			assert.Equal(t, int64(300000), payload.PollRate)

			mu.Lock()
			createdNames = append(createdNames, payload.Name)
			sequence := len(createdNames)
			id := "aws-external-primary"
			externalID := "external-primary"
			account := "arn:aws:iam::111111111111:root"
			if sequence == 2 {
				id = "aws-external-replacement"
				externalID = "external-replacement"
				account = "arn:aws:iam::222222222222:root"
			}
			stored[id] = integration.AwsCloudWatchIntegration{
				Id: id, Name: payload.Name, Type: "AWSCloudWatch", AuthMethod: integration.EXTERNAL_ID,
				ExternalId: externalID, SfxAwsAccountArn: account, PollRate: payload.PollRate,
			}
			mu.Unlock()
			writeStored(w, id)
		}),
		"GET /v2/integration/aws-external-primary": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeStored(w, "aws-external-primary")
		}),
		"GET /v2/integration/aws-external-replacement": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeStored(w, "aws-external-replacement")
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAWSExternal)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/aws_external_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_aws_external_integration.test", "id", "aws-external-primary"),
				testresource.TestCheckResourceAttr("signalfx_aws_external_integration.test", "name", "Primary AWS external"),
				testresource.TestCheckResourceAttr("signalfx_aws_external_integration.test", "external_id", "external-primary"),
				testresource.TestCheckResourceAttr("signalfx_aws_external_integration.test", "signalfx_aws_account", "arn:aws:iam::111111111111:root"),
			)},
			{ConfigFile: config.StaticFile("testdata/aws_external_create.tf"), PlanOnly: true},
			{ConfigFile: config.StaticFile("testdata/aws_external_replace.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_aws_external_integration.test", "id", "aws-external-replacement"),
				testresource.TestCheckResourceAttr("signalfx_aws_external_integration.test", "name", "Replacement AWS external"),
				testresource.TestCheckResourceAttr("signalfx_aws_external_integration.test", "external_id", "external-replacement"),
				testresource.TestCheckResourceAttr("signalfx_aws_external_integration.test", "signalfx_aws_account", "arn:aws:iam::222222222222:root"),
			)},
			{ConfigFile: config.StaticFile("testdata/aws_external_replace.tf"), PlanOnly: true},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"Primary AWS external", "Replacement AWS external"}, createdNames)
}

func TestResourceAWSExternalRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := integration.AwsCloudWatchIntegration{
		Id: "aws-external-primary", Name: "Primary AWS external", Type: "AWSCloudWatch", AuthMethod: integration.EXTERNAL_ID,
		ExternalId: "external-primary", SfxAwsAccountArn: "arn:aws:iam::111111111111:root", PollRate: 300000,
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(current)
		}),
		"GET /v2/integration/aws-external-primary": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				_ = json.NewEncoder(w).Encode(current)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAWSExternal)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/aws_external_create.tf")},
			{ConfigFile: config.StaticFile("testdata/aws_external_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceAWSExternalCreateErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		error  *regexp.Regexp
	}{
		{name: "administrator guidance", status: http.StatusUnauthorized, error: regexp.MustCompile(adminTokenHelp)},
		{name: "server failure", status: http.StatusBadGateway, error: regexp.MustCompile(`status code 502`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			endpoints := map[string]http.Handler{
				"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "failed", test.status)
				}),
			}
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAWSExternal)),
				Steps: []testresource.TestStep{{
					ConfigFile:  config.StaticFile("testdata/aws_external_create.tf"),
					ExpectError: test.error,
				}},
			})
		})
	}
}
