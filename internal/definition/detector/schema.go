// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/signalfx/signalfx-go/detector"
	"go.uber.org/multierr"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/check"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/rule"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/visual"
)

func newSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the detector",
		},
		"program_text": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "Signalflow program text for the detector. More info at \"https://developers.signalfx.com/docs/signalflow-overview\"",
			ValidateFunc: validation.StringLenBetween(1, 50000),
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Description of the detector",
		},
		"timezone": {
			Type:             schema.TypeString,
			Optional:         true,
			Default:          "UTC",
			ValidateDiagFunc: check.TimeZoneLocation(),
			Description:      "The property value is a string that denotes the geographic region associated with the time zone, (e.g. Australia/Sydney)",
		},
		"max_delay": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      0,
			Description:  "Maximum time (in seconds) to wait for late datapoints. Max value is 900 (15m)",
			ValidateFunc: validation.IntBetween(0, 900),
		},
		"min_delay": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      0,
			Description:  "Minimum time (in seconds) for the computation to wait even if the datapoints are arriving in a timely fashion. Max value is 900 (15m)",
			ValidateFunc: validation.IntBetween(0, 900),
		},
		"show_data_markers": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "(true by default) When true, markers will be drawn for each datapoint within the visualization.",
		},
		"show_event_lines": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "(false by default) When true, vertical lines will be drawn for each triggered event within the visualization.",
		},
		"disable_sampling": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "(false by default) When false, samples a subset of the output MTS in the visualization.",
		},
		"time_range": {
			Type:          schema.TypeInt,
			Optional:      true,
			Default:       3600,
			Description:   "Seconds to display in the visualization. This is a rolling range from the current time. Example: 3600 = `-1h`. Defaults to 3600",
			ConflictsWith: []string{"start_time", "end_time"},
		},
		"start_time": {
			Type:          schema.TypeInt,
			Optional:      true,
			ConflictsWith: []string{"time_range"},
			Description:   "Seconds since epoch. Used for visualization",
			ValidateFunc:  validation.IntAtLeast(0),
		},
		"end_time": {
			Type:          schema.TypeInt,
			Optional:      true,
			ConflictsWith: []string{"time_range"},
			Description:   "Seconds since epoch. Used for visualization",
			ValidateFunc:  validation.IntAtLeast(0),
		},
		"tags": {
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Tags associated with the detector",
		},
		"teams": {
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Team IDs to associate the detector to",
		},
		"rule": {
			Type:        schema.TypeSet,
			Required:    true,
			Description: "Set of rules used for alerting",
			Elem: &schema.Resource{
				SchemaFunc: rule.NewSchema,
			},
			Set: rule.Hash,
		},
		"authorized_writer_teams": {
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Team IDs that have write access to this dashboard",
		},
		"authorized_writer_users": {
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "User IDs that have write access to this dashboard",
		},
		"viz_options": {
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "Plot-level customization options, associated with a publish statement",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"label": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The label used in the publish statement that displays the plot (metric time series data) you want to customize",
					},
					"color": {
						Type:             schema.TypeString,
						Optional:         true,
						Description:      "Color to use",
						ValidateDiagFunc: check.ColorName(),
					},
					"display_name": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.",
					},
					"value_unit": {
						Type:             schema.TypeString,
						Optional:         true,
						ValidateDiagFunc: check.ValueUnit(),
						Description:      "A unit to attach to this plot. Units support automatic scaling (eg thousands of bytes will be displayed as kilobytes)",
					},
					"value_prefix": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An arbitrary prefix to display with the value of this plot",
					},
					"value_suffix": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An arbitrary suffix to display with the value of this plot",
					},
				},
			},
		},
		"label_resolutions": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "Resolutions of the detector alerts in milliseconds that indicate how often data is analyzed to determine if an alert should be triggered",
			Elem: &schema.Schema{
				Type: schema.TypeInt,
			},
		},
		"url": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "URL of the detector",
		},
		"detector_origin": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "Standard",
			Description:  "Indicates how a detector was created",
			ValidateFunc: validation.StringInSlice([]string{"Standard", "AutoDetectCustomization"}, false),
		},
		"parent_detector_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "ID of the parent AutoDetect detector from which this detector is customized and created. This property is required for detectors with detector_origin of type AutoDetectCustomization.",
		},
	}
}

func decodeTerraform(rd tfext.Values) (*detector.Detector, error) {
	d := &detector.Detector{
		Id:                rd.Id(),
		Name:              rd.Get("name").(string),
		Description:       rd.Get("description").(string),
		ProgramText:       rd.Get("program_text").(string),
		TimeZone:          rd.Get("timezone").(string),
		DetectorOrigin:    rd.Get("detector_origin").(string),
		ParentDetectorId:  rd.Get("parent_detector_id").(string),
		AuthorizedWriters: &detector.AuthorizedWriters{},
		//nolint:gosec // Overflow is not possible from config
		MinDelay: common.AsPointer(int32(rd.Get("min_delay").(int)) * 1000),
		//nolint:gosec // Overflow is not possible from config
		MaxDelay: common.AsPointer(int32(rd.Get("max_delay").(int)) * 1000),
		VisualizationOptions: &detector.Visualization{
			DisableSampling: rd.Get("disable_sampling").(bool),
			ShowDataMarkers: rd.Get("show_data_markers").(bool),
			ShowEventLines:  rd.Get("show_event_lines").(bool),
		},
	}

	if tr, ok := rd.GetOk("time_range"); ok {
		d.VisualizationOptions.Time = &detector.Time{
			Range: common.AsPointer(int64(tr.(int)) * 1000),
			Type:  "relative",
		}
	}

	if rd.HasChanges("start_time", "end_time") {
		d.VisualizationOptions.Time = &detector.Time{
			Type:  "absolute",
			Start: common.AsPointer(int64(rd.Get("start_time").(int)) * 1000),
			End:   common.AsPointer(int64(rd.Get("end_time").(int)) * 1000),
		}

	}

	for field, ref := range map[string]*[]string{
		"teams":                   &d.Teams,
		"tags":                    &d.Tags,
		"authorized_writer_teams": &d.AuthorizedWriters.Teams,
		"authorized_writer_users": &d.AuthorizedWriters.Users,
	} {
		if values, exist := rd.GetOk(field); exist {
			for _, v := range values.(*schema.Set).List() {
				(*ref) = append((*ref), v.(string))
			}
		}
	}

	rules, err := rule.DecodeTerraform(rd)
	if err != nil {
		return nil, err
	}
	d.Rules = rules

	palette := visual.NewColorPalette()
	for _, data := range rd.Get("viz_options").(*schema.Set).List() {
		viz := data.(map[string]any)
		opt := &detector.PublishLabelOptions{
			Label:       viz["label"].(string),
			DisplayName: viz["display_name"].(string),
			ValueUnit:   viz["value_unit"].(string),
			ValuePrefix: viz["value_prefix"].(string),
			ValueSuffix: viz["value_suffix"].(string),
		}

		if idx, ok := palette.ColorIndex(viz["color"].(string)); ok {
			opt.PaletteIndex = common.AsPointer(idx)
		}

		d.VisualizationOptions.PublishLabelOptions = append(d.VisualizationOptions.PublishLabelOptions, opt)
	}

	return d, nil
}

func encodeTerraform(dt *detector.Detector, rd *schema.ResourceData) error {
	rd.SetId(dt.Id)

	errs := multierr.Combine(
		rd.Set("name", dt.Name),
		rd.Set("description", dt.Description),
		rd.Set("timezone", dt.TimeZone),
		rd.Set("program_text", dt.ProgramText),
		rd.Set("detector_origin", dt.DetectorOrigin),
		rd.Set("parent_detector_id", dt.ParentDetectorId),
		rd.Set("teams", dt.Teams),
		rd.Set("tags", dt.Tags),
		rd.Set("label_resolutions", dt.LabelResolutions),
	)
	// We divide by 1000 because the API uses millis, but this provider uses
	// seconds
	if dt.MinDelay != nil {
		errs = multierr.Append(errs, rd.Set("min_delay", *dt.MinDelay/1000))
	}
	if dt.MaxDelay != nil {
		errs = multierr.Append(errs, rd.Set("max_delay", *dt.MaxDelay/1000))
	}
	if auth := dt.AuthorizedWriters; auth != nil {
		errs = multierr.Append(errs, rd.Set("authorized_writer_teams", tfext.NewSchemaSet(schema.HashString, auth.Teams)))
		errs = multierr.Append(errs, rd.Set("authorized_writer_users", tfext.NewSchemaSet(schema.HashString, auth.Users)))
	}

	if viz := dt.VisualizationOptions; viz != nil {
		errs = multierr.Append(errs, multierr.Combine(
			rd.Set("disable_sampling", dt.VisualizationOptions.DisableSampling),
			rd.Set("show_data_markers", dt.VisualizationOptions.ShowDataMarkers),
			rd.Set("show_event_lines", dt.VisualizationOptions.ShowEventLines),
		))
		if t := viz.Time; t != nil {
			switch {
			case t.Start != nil && t.End != nil:
				errs = multierr.Append(errs, rd.Set("start_time", *t.Start))
				errs = multierr.Append(errs, rd.Set("end_time", *t.End))
			case t.Range != nil:
				errs = multierr.Append(errs, rd.Set("time_range", *t.Range))
			}
		}
		labels := make([]map[string]any, 0, len(viz.PublishLabelOptions))
		palette := visual.NewColorPalette()

		for _, opts := range viz.PublishLabelOptions {
			color := ""
			if pi := opts.PaletteIndex; pi != nil {
				if name, ok := palette.IndexColorName(*pi); ok {
					color = name
				}
			}

			labels = append(labels, map[string]any{
				"label":        opts.Label,
				"display_name": opts.DisplayName,
				"value_unit":   opts.ValueUnit,
				"value_suffix": opts.ValueSuffix,
				"value_prefix": opts.ValuePrefix,
				"color":        color,
			})

		}
		errs = multierr.Append(errs, rd.Set("viz_options", labels))
	}

	return multierr.Append(errs, rule.EncodeTerraform(dt.Rules, rd))
}
