// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

func TestValidateAWSIntegrationDiff(t *testing.T) {
	t.Parallel()

	baseConfig := map[string]interface{}{
		"integration_id": "integration-id",
		"enabled":        true,
		"regions":        []interface{}{"us-east-1"},
	}

	for _, tc := range []struct {
		name    string
		values  map[string]interface{}
		wantErr string
	}{
		{
			name: "both enabled",
			values: map[string]interface{}{
				"import_cloud_watch":      true,
				"use_metric_streams_sync": true,
			},
			wantErr: "`import_cloud_watch` and `use_metric_streams_sync` cannot both be true; set one of them to false",
		},
		{
			name: "cloud watch only",
			values: map[string]interface{}{
				"import_cloud_watch": true,
			},
		},
		{
			name: "metric streams only",
			values: map[string]interface{}{
				"use_metric_streams_sync": true,
			},
		},
		{
			name: "both explicitly disabled",
			values: map[string]interface{}{
				"import_cloud_watch":      false,
				"use_metric_streams_sync": false,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := make(map[string]interface{}, len(baseConfig)+2)
			for key, value := range baseConfig {
				config[key] = value
			}
			for key, value := range tc.values {
				config[key] = value
			}

			_, err := integrationAWSResource().Diff(
				context.Background(),
				nil,
				terraform.NewResourceConfigRaw(config),
				nil,
			)

			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantErr)
			}
		})
	}
}

func TestGetPayloadAWSIntegrationRejectsConflictingImportModes(t *testing.T) {
	t.Parallel()

	data := schema.TestResourceDataRaw(t, integrationAWSResource().Schema, map[string]interface{}{
		"import_cloud_watch":      true,
		"use_metric_streams_sync": true,
	})

	payload, err := getPayloadAWSIntegration(data)

	assert.Nil(t, payload)
	assert.EqualError(t, err, "`import_cloud_watch` and `use_metric_streams_sync` cannot both be true; set one of them to false")
}
