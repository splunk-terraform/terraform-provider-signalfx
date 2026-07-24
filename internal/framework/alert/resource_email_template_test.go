// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/emailtemplate"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceEmailTemplateMetadata(t *testing.T) {
	t.Parallel()

	r := NewResourceEmailTemplate()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_email_template", resp.TypeName)
}

func TestResourceEmailTemplateSchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourceEmailTemplate(), emailTemplateModel{
		To:            types.ListNull(types.StringType),
		Cc:            types.ListNull(types.StringType),
		Bcc:           types.ListNull(types.StringType),
		CustomHeaders: types.MapNull(types.StringType),
	}))
}

func TestResourceEmailTemplateUnitTest(t *testing.T) {
	t.Parallel()

	var (
		current emailtemplate.EmailTemplate
		mu      sync.Mutex
	)

	endpoints := map[string]http.Handler{
		"POST /v2/alert/emailtemplate": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var data emailtemplate.EmailTemplate
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			assert.Equal(t, "Detector Alert Email", data.Name)
			assert.Equal(t, "Triggered: {{{detectorName}}}", data.TriggerSubject)
			assert.Equal(t, "Alert body {{{messageTitle}}}", data.TriggerBody)
			assert.Equal(t, "Resolved: {{{detectorName}}}", data.ResolvedSubject)
			assert.Equal(t, "Resolved body {{{messageTitle}}}", data.ResolvedBody)
			assert.Equal(t, []string{"primary@example.com"}, data.To)
			assert.Equal(t, []string{"team@example.com"}, data.Cc)
			assert.Equal(t, []string{"audit@example.com"}, data.Bcc)
			assert.Equal(t, map[string]string{"X-SFX-Template": "detector"}, data.CustomHeaders)

			data.Id = "template-id"
			data.CreatedOnMs = 1720000000000
			data.CreatedBy = "creator@example.com"
			data.UpdatedOnMs = 1720000000000
			data.UpdatedBy = "creator@example.com"

			mu.Lock()
			current = data
			mu.Unlock()

			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}),
		"GET /v2/alert/emailtemplate/template-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			data := current
			mu.Unlock()

			if err := json.NewEncoder(w).Encode(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}),
		"PUT /v2/alert/emailtemplate/template-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var data emailtemplate.EmailTemplate
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			assert.Equal(t, "Detector Alert Email Updated", data.Name)
			assert.Equal(t, []string{"primary@example.com", "secondary@example.com"}, data.To)
			assert.Equal(t, []string{"team@example.com"}, data.Cc)
			assert.Empty(t, data.Bcc)
			assert.Equal(t, map[string]string{"X-SFX-Template": "detector-updated"}, data.CustomHeaders)

			data.Id = "template-id"
			data.CreatedOnMs = 1720000000000
			data.CreatedBy = "creator@example.com"
			data.UpdatedOnMs = 1720000001000
			data.UpdatedBy = "updater@example.com"

			mu.Lock()
			current = data
			mu.Unlock()

			if err := json.NewEncoder(w).Encode(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}),
		"DELETE /v2/alert/emailtemplate/template-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(
		t,
		testresource.TestCase{
			IsUnitTest: true,
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.RequireAbove(tfversion.Version0_12_26),
			},
			ProtoV5ProviderFactories: fwtest.NewMockProto5Server(
				t,
				endpoints,
				fwtest.WithMockResources(NewResourceEmailTemplate),
			),
			Steps: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_email_template.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "id", "template-id"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "name", "Detector Alert Email"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "to.0", "primary@example.com"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "cc.0", "team@example.com"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "bcc.0", "audit@example.com"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "custom_headers.X-SFX-Template", "detector"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "created_by", "creator@example.com"),
					),
				},
				{
					ConfigFile: config.StaticFile("testdata/01_email_template_updated.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "id", "template-id"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "name", "Detector Alert Email Updated"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "to.#", "2"),
						testresource.TestCheckNoResourceAttr("signalfx_email_template.test", "bcc.#"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "custom_headers.X-SFX-Template", "detector-updated"),
						testresource.TestCheckResourceAttr("signalfx_email_template.test", "updated_by", "updater@example.com"),
					),
				},
			},
		},
	)
}
