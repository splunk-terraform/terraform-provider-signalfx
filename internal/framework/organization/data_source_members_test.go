// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fworganization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/organization"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestDataSourceMembersMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewDataSourceMembers()
	metadata := &datasource.MetadataResponse{}
	implementation.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_organization_members", metadata.TypeName)

	model := dataSourceMembersModel{
		Emails: types.ListValueMust(types.StringType, nil),
		Users:  types.ListValueMust(types.StringType, nil),
	}
	assert.NoError(t, fwtest.DataSourceSchemaValidate(implementation, model))

	schemaResponse := &datasource.SchemaResponse{}
	implementation.Schema(context.Background(), datasource.SchemaRequest{}, schemaResponse)
	assert.Len(t, schemaResponse.Schema.Attributes["emails"].(schema.ListAttribute).Validators, 1)
}

func TestDataSourceMembersID(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "4iwxhwgx4d6y", organizationMembersID([]string{"alice@example.com", "bob@example.com"}))
	assert.NotEqual(t,
		organizationMembersID([]string{"alice@example.com", "bob@example.com"}),
		organizationMembersID([]string{"bob@example.com", "alice@example.com"}),
		"the legacy ID is order-sensitive",
	)
}

func TestEmailListValidator(t *testing.T) {
	t.Parallel()
	implementation := emailListValidator{}
	assert.Equal(t, "each value must be a valid email address", implementation.Description(context.Background()))
	assert.Equal(t, implementation.Description(context.Background()), implementation.MarkdownDescription(context.Background()))

	for _, test := range []struct {
		name  string
		value types.List
		error bool
	}{
		{name: "null", value: types.ListNull(types.StringType)},
		{name: "unknown", value: types.ListUnknown(types.StringType)},
		{name: "valid", value: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("alice@example.com"), types.StringValue("Bob <bob@example.com>"),
		})},
		{name: "invalid", value: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("alice@example.com"), types.StringValue("not-an-email"),
		}), error: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := &validator.ListResponse{}
			implementation.ValidateList(context.Background(), validator.ListRequest{
				Path: path.Root("emails"), ConfigValue: test.value,
			}, response)
			assert.Equal(t, test.error, response.Diagnostics.HasError())
			if test.error {
				diagnostic, ok := response.Diagnostics[0].(diag.DiagnosticWithPath)
				require.True(t, ok)
				assert.Equal(t, `emails[1]`, diagnostic.Path().String())
			}
		})
	}
}

func TestDataSourceMembersRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &DataSourceMembers{}
	schemaResponse := &datasource.SchemaResponse{}
	implementation.Schema(ctx, datasource.SchemaRequest{}, schemaResponse)
	response := &datasource.ReadResponse{}
	implementation.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Bool, true), Schema: schemaResponse.Schema,
	}}, response)
	assert.True(t, response.Diagnostics.HasError())
}

func TestDataSourceMembersMockedRead(t *testing.T) {
	var mu sync.Mutex
	calls := make(map[string]int)
	endpoints := map[string]http.Handler{
		"GET /v2/organization/member": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, "1000", query.Get("limit"))
			assert.Equal(t, "-sf_timestamp", query.Get("orderBy"))
			emailQuery := query.Get("query")
			offset, err := strconv.Atoi(query.Get("offset"))
			if err != nil {
				t.Errorf("parse offset: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			calls[emailQuery+":"+strconv.Itoa(offset)]++
			mu.Unlock()

			response := &organization.MemberSearchResults{}
			switch {
			case emailQuery == "email:alice@example.com" && offset == 0:
				response.Count = 1001
				response.Results = []*organization.Member{{Id: "alice-primary"}, nil}
			case emailQuery == "email:alice@example.com" && offset == 1000:
				response.Count = 1001
				response.Results = []*organization.Member{{Id: "alice-secondary"}}
			case emailQuery == "email:bob@example.com" && offset == 0:
				response.Count = 1
				response.Results = []*organization.Member{{Id: "bob"}}
			default:
				http.Error(w, "unexpected query: "+r.URL.RawQuery, http.StatusBadRequest)
				return
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Errorf("write member response: %v", err)
			}
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockDataSources(NewDataSourceMembers)),
		Steps: []testresource.TestStep{{
			ConfigFile: config.StaticFile("testdata/members.tf"),
			Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("data.signalfx_organization_members.test", "id", "4iwxhwgx4d6y"),
				testresource.TestCheckResourceAttr("data.signalfx_organization_members.test", "users.#", "3"),
				testresource.TestCheckResourceAttr("data.signalfx_organization_members.test", "users.0", "alice-primary"),
				testresource.TestCheckResourceAttr("data.signalfx_organization_members.test", "users.1", "alice-secondary"),
				testresource.TestCheckResourceAttr("data.signalfx_organization_members.test", "users.2", "bob"),
			),
		}},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, map[string]int{
		"email:alice@example.com:0":    3,
		"email:alice@example.com:1000": 3,
		"email:bob@example.com:0":      3,
	}, calls)
}

func TestDataSourceMembersEmptyResults(t *testing.T) {
	endpoints := map[string]http.Handler{
		"GET /v2/organization/member": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, url.Values{
				"limit": {"1000"}, "offset": {"0"}, "orderBy": {"-sf_timestamp"}, "query": {"email:nobody@example.com"},
			}, r.URL.Query())
			_ = json.NewEncoder(w).Encode(&organization.MemberSearchResults{Count: 10})
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockDataSources(NewDataSourceMembers)),
		Steps: []testresource.TestStep{{
			Config: `data "signalfx_organization_members" "test" { emails = ["nobody@example.com"] }`,
			Check:  testresource.TestCheckResourceAttr("data.signalfx_organization_members.test", "users.#", "0"),
		}},
	})
}

func TestDataSourceMembersMockedErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		config    string
		endpoints map[string]http.Handler
		error     *regexp.Regexp
	}{
		{
			name:   "API error",
			config: `data "signalfx_organization_members" "test" { emails = ["alice@example.com"] }`,
			endpoints: map[string]http.Handler{"GET /v2/organization/member": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			})},
			error: regexp.MustCompile(`status code 401`),
		},
		{name: "missing emails", config: `data "signalfx_organization_members" "test" {}`, error: regexp.MustCompile(`argument "emails" is required`)},
		{name: "invalid email", config: `data "signalfx_organization_members" "test" { emails = ["invalid"] }`, error: regexp.MustCompile(`Invalid email address`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, test.endpoints, fwtest.WithMockDataSources(NewDataSourceMembers)),
				Steps:                    []testresource.TestStep{{Config: test.config, PlanOnly: true, ExpectError: test.error}},
			})
		})
	}
}
