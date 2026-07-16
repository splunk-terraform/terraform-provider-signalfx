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

func TestResourceAWSTokenMetadata(t *testing.T) {
	t.Parallel()

	implementation := NewResourceAWSToken()
	response := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, response)

	assert.Equal(t, "signalfx_aws_token_integration", response.TypeName)
}

func TestResourceAWSTokenSchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourceAWSToken(), resourceAWSTokenModel{}))

	response := &resource.SchemaResponse{}
	NewResourceAWSToken().Schema(context.Background(), resource.SchemaRequest{}, response)
	name, ok := response.Schema.Attributes["name"].(schema.StringAttribute)
	require.True(t, ok)
	assert.Len(t, name.PlanModifiers, 1)
	for _, attributeName := range []string{"token_id", "signalfx_aws_account"} {
		attribute, attributeOK := response.Schema.Attributes[attributeName].(schema.StringAttribute)
		require.True(t, attributeOK)
		assert.True(t, attribute.Computed)
		assert.True(t, attribute.Sensitive)
	}
}

func TestResourceAWSTokenModel(t *testing.T) {
	t.Parallel()

	model := resourceAWSTokenModel{
		awsBootstrapModel: awsBootstrapModel{
			ID:                 types.StringValue("aws-id"),
			Name:               types.StringValue("Primary AWS token"),
			SignalFxAWSAccount: types.StringValue("arn:aws:iam::111111111111:root"),
		},
		TokenID: types.StringUnknown(),
	}

	assert.Equal(t, &integration.AwsCloudWatchIntegration{
		Type:       "AWSCloudWatch",
		AuthMethod: integration.SECURITY_TOKEN,
		Name:       "Primary AWS token",
		PollRate:   300000,
	}, model.awsIntegration(integration.SECURITY_TOKEN))

	model.updateFromAPI(nil, true)
	assert.True(t, model.TokenID.IsUnknown())

	model.updateFromAPI(&integration.AwsCloudWatchIntegration{
		Id: "updated-id", Name: "Updated by API", AuthMethod: integration.SECURITY_TOKEN,
		Token: "api-token-value", SfxAwsAccountArn: "arn:aws:iam::222222222222:root",
	}, true)
	assert.Equal(t, types.StringValue("updated-id"), model.ID)
	assert.Equal(t, types.StringValue("Updated by API"), model.Name)
	assert.Equal(t, types.StringValue(""), model.TokenID, "the legacy token_id was not populated from the API token field")
	assert.Equal(t, types.StringValue("arn:aws:iam::222222222222:root"), model.SignalFxAWSAccount)

	model.TokenID = types.StringValue("preserved-token-id")
	model.updateFromAPI(&integration.AwsCloudWatchIntegration{
		Id: "ignored-id", Name: "Ignored read name", AuthMethod: integration.SECURITY_TOKEN,
		SfxAwsAccountArn: "arn:aws:iam::333333333333:root",
	}, false)
	assert.Equal(t, types.StringValue("updated-id"), model.ID)
	assert.Equal(t, types.StringValue("Updated by API"), model.Name)
	assert.Equal(t, types.StringValue("preserved-token-id"), model.TokenID)
	assert.Equal(t, types.StringValue("arn:aws:iam::333333333333:root"), model.SignalFxAWSAccount)
}

func TestResourceAWSTokenRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	implementation := &ResourceAWSToken{}
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

func TestResourceAWSTokenMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	createdNames := make([]string, 0, 2)
	stored := map[string]integration.AwsCloudWatchIntegration{}

	writeStored := func(w http.ResponseWriter, id string) {
		mu.Lock()
		current := stored[id]
		mu.Unlock()
		if err := json.NewEncoder(w).Encode(current); err != nil {
			t.Errorf("write AWS token response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload integration.AwsCloudWatchIntegration
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode AWS token payload: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assert.Equal(t, integration.Type("AWSCloudWatch"), payload.Type)
			assert.Equal(t, integration.SECURITY_TOKEN, payload.AuthMethod)
			assert.Equal(t, int64(300000), payload.PollRate)

			mu.Lock()
			createdNames = append(createdNames, payload.Name)
			sequence := len(createdNames)
			id := "aws-token-primary"
			account := "arn:aws:iam::111111111111:root"
			if sequence == 2 {
				id = "aws-token-replacement"
				account = "arn:aws:iam::222222222222:root"
			}
			stored[id] = integration.AwsCloudWatchIntegration{
				Id: id, Name: payload.Name, Type: "AWSCloudWatch", AuthMethod: integration.SECURITY_TOKEN,
				Token: "api-token-value", SfxAwsAccountArn: account, PollRate: payload.PollRate,
			}
			mu.Unlock()
			writeStored(w, id)
		}),
		"GET /v2/integration/aws-token-primary": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeStored(w, "aws-token-primary")
		}),
		"GET /v2/integration/aws-token-replacement": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeStored(w, "aws-token-replacement")
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAWSToken)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/aws_token_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_aws_token_integration.test", "id", "aws-token-primary"),
				testresource.TestCheckResourceAttr("signalfx_aws_token_integration.test", "name", "Primary AWS token"),
				testresource.TestCheckResourceAttr("signalfx_aws_token_integration.test", "token_id", ""),
				testresource.TestCheckResourceAttr("signalfx_aws_token_integration.test", "signalfx_aws_account", "arn:aws:iam::111111111111:root"),
			)},
			{ConfigFile: config.StaticFile("testdata/aws_token_create.tf"), PlanOnly: true},
			{ConfigFile: config.StaticFile("testdata/aws_token_replace.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_aws_token_integration.test", "id", "aws-token-replacement"),
				testresource.TestCheckResourceAttr("signalfx_aws_token_integration.test", "name", "Replacement AWS token"),
				testresource.TestCheckResourceAttr("signalfx_aws_token_integration.test", "token_id", ""),
				testresource.TestCheckResourceAttr("signalfx_aws_token_integration.test", "signalfx_aws_account", "arn:aws:iam::222222222222:root"),
			)},
			{ConfigFile: config.StaticFile("testdata/aws_token_replace.tf"), PlanOnly: true},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"Primary AWS token", "Replacement AWS token"}, createdNames)
}

func TestResourceAWSTokenRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := integration.AwsCloudWatchIntegration{
		Id: "aws-token-primary", Name: "Primary AWS token", Type: "AWSCloudWatch", AuthMethod: integration.SECURITY_TOKEN,
		Token: "api-token-value", SfxAwsAccountArn: "arn:aws:iam::111111111111:root", PollRate: 300000,
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(current)
		}),
		"GET /v2/integration/aws-token-primary": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAWSToken)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/aws_token_create.tf")},
			{ConfigFile: config.StaticFile("testdata/aws_token_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceAWSTokenCreateErrors(t *testing.T) {
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
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAWSToken)),
				Steps: []testresource.TestStep{{
					ConfigFile:  config.StaticFile("testdata/aws_token_create.tf"),
					ExpectError: test.error,
				}},
			})
		})
	}
}
