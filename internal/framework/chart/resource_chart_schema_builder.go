package fwchart

import (
	"maps"
	"math"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	fwvalidator "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/validator"
)

type SchemaBuilder struct {
	applies []map[string]schema.Attribute
}

func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{}
}

func (sb *SchemaBuilder) Build() map[string]schema.Attribute {
	attributes := make(map[string]schema.Attribute)
	for _, apply := range sb.applies {
		maps.Copy(attributes, apply)
	}
	return attributes
}

func (sb *SchemaBuilder) WithCommonChartBase() *SchemaBuilder {
	return sb.WithRequiredProgramText().
		WithOptionalUnitPrefix().
		WithOptionalMaxDelay().
		WithOptionalTimeZone().
		WithOptionalRefreshInterval().
		WithOptionalDisableSampling()
}

func (sb *SchemaBuilder) WithOptionalAxes() *SchemaBuilder {
	watermark := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"value": schema.Float64Attribute{
				Required:    true,
				Description: "Value for the watermark.",
			},
			"label": schema.StringAttribute{
				Optional:    true,
				Description: "Label for the watermark.",
			},
		},
	}

	axis := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"min_value": schema.Float64Attribute{
				Optional:    true,
				Description: "Minimum value for the axis.",
			},
			"max_value": schema.Float64Attribute{
				Optional:    true,
				Description: "Maximum value for the axis.",
			},
			"label": schema.StringAttribute{
				Optional:    true,
				Description: "Label for the axis.",
			},
			"high_watermark": watermark,
			"low_watermark":  watermark,
		},
	}
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"axis_right": axis,
				"axis_left":  axis,
				"axes_precision": schema.Int64Attribute{
					Optional: true,
					Computed: true,
					Default:  int64default.StaticInt64(3),
				},
				"axes_include_zero": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Force the y-axes to include zero in the chart.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalColorBy() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"color_by": schema.StringAttribute{
					Optional:    true,
					Description: "Color by attribute for the chart.",
					Default:     stringdefault.StaticString("Dimension"),
					Validators: []validator.String{
						stringvalidator.OneOf("Metric", "Dimension", "Scale"),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalColorRange() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"color_range": schema.ListNestedAttribute{
					Optional:    true,
					Description: "Color range for the chart.",
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"color": schema.StringAttribute{
								Required: true,
							},
							"min_value": schema.Float32Attribute{
								Optional:    true,
								Description: "Minimum value for the color range.",
							},
							"max_value": schema.Float32Attribute{
								Optional:    true,
								Description: "Maximum value for the color range.",
							},
						},
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalColorScale() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"color_scale": schema.ListNestedAttribute{
					Optional:    true,
					Description: "Color scale for the chart.",
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"color": schema.StringAttribute{
								Required:    true,
								Description: "Color for the scale.",
								Validators:  []validator.String{
									// TODO(MovieStoreGuy): Add color validator for color scale.
								},
							},
							"gt": schema.Float32Attribute{
								Optional: true,
								Computed: true,
								Default:  float32default.StaticFloat32(math.MaxFloat32),
							},
							"gte": schema.Float32Attribute{
								Optional: true,
								Computed: true,
								Default:  float32default.StaticFloat32(math.MaxFloat32),
							},
							"lt": schema.Float32Attribute{
								Optional: true,
								Computed: true,
								Default:  float32default.StaticFloat32(-math.MaxFloat32),
							},
							"lte": schema.Float32Attribute{
								Optional: true,
								Computed: true,
								Default:  float32default.StaticFloat32(-math.MaxFloat32),
							},
						},
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalDisableSampling() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"disable_sampling": schema.BoolAttribute{
					Optional:    true,
					Description: "Disables sampling for the chart.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalEventOptions() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"event_options": schema.ListNestedAttribute{
					Optional: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"label": schema.StringAttribute{
								Required:    true,
								Description: "Label for the event option.",
							},
							"color": schema.StringAttribute{
								Optional:    true,
								Description: "Color for the event option.",
								Validators:  []validator.String{
									// TODO(MovieStoreGuy): Add color validator for event options.
								},
							},
							"display_name": schema.StringAttribute{
								Optional:    true,
								Description: "Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.",
							},
						},
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalShowEventLines() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"show_event_lines": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Enables showing event lines in the chart.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalGroupBy() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"group_by": schema.ListAttribute{
					Optional:    true,
					Description: "List of attributes to group by in the chart.",
					ElementType: types.StringType,
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalHideMissingValues() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"hide_missing_values": schema.BoolAttribute{
					Optional:    true,
					Description: "Hides missing values in the chart.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalHideTimeStamps() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"hide_timestamp": schema.BoolAttribute{
					Optional:    true,
					Description: "Hides time stamps in the chart.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalHistogram() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"histogram_options": schema.SingleNestedAttribute{
					Optional: true,
					Attributes: map[string]schema.Attribute{
						"color_theme": schema.StringAttribute{
							Optional:    true,
							Description: "Color theme for the histogram.",
						},
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalLegend() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"legend_options_fields": schema.ListNestedAttribute{
					Optional: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"property": schema.StringAttribute{
								Required:    true,
								Description: "The name of the property to hide or show in the data table.",
							},
							"enabled": schema.BoolAttribute{
								Required:    true,
								Description: "Determines if the property is displayed in the data table.",
							},
						},
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalMaxDelay() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"max_delay": schema.Int64Attribute{
					Optional:    true,
					Description: "Maximum delay in seconds for the chart.",
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalMaxPrecision() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"max_precision": schema.Int64Attribute{
					Optional:    true,
					Description: "Maximum precision in seconds for the chart.",
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalMinResolution() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"min_resolution": schema.Int64Attribute{
					Optional:    true,
					Description: "Minimum resolution in seconds for the chart.",
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalPlotType() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"plot_type": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Plot type for the chart.",
					Default:     stringdefault.StaticString("LineChart"),
					Validators: []validator.String{
						stringvalidator.OneOf("AreaChart", "ColumnChart", "Histogram", "LineChart"),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalRefreshInterval() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"refresh_interval": schema.Int64Attribute{
					Optional:    true,
					Description: "Refresh interval in seconds for the chart.",
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalSecondaryVisualization() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"secondary_visualization": schema.StringAttribute{
					Optional:    true,
					Description: "Secondary visualization type for the chart.",
					Validators: []validator.String{
						stringvalidator.OneOf("None", "Radial", "Linear", "SparkLine"),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalShowDataMarkers() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"show_data_markers": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Enables showing data points in the chart with circluar markers.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalShowSparkLine() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"show_spark_line": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Enables showing trend line below the current chart value.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalSortBy() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"sort_by": schema.StringAttribute{
					Optional:    true,
					Description: "List of attributes to sort by in the chart.",
					Validators: []validator.String{
						fwvalidator.NewSortString(),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalStacked() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"stacked": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Enables stacked visualization for the chart.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalStartEndTime() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"start_time": schema.Int64Attribute{
					Optional:    true,
					Description: "Seconds since epoch to start the visualization.",
					Validators: []validator.Int64{
						int64validator.ConflictsWith(path.MatchRoot("time_range")),
						int64validator.AlsoRequires(path.MatchRoot("end_time")),
					},
				},
				"end_time": schema.Int64Attribute{
					Optional:    true,
					Description: "Seconds since epoch to end the visualization.",
					Validators: []validator.Int64{
						int64validator.ConflictsWith(path.MatchRoot("time_range")),
						int64validator.AlsoRequires(path.MatchRoot("start_time")),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalTimeZone() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"timezone": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Timezone for the chart.",
					Default:     stringdefault.StaticString("UTC"),
					Validators: []validator.String{
						fwvalidator.NewTimeZoneValidator(),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalTimeRange() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"time_range": schema.Int64Attribute{
					Optional:    true,
					Description: "Seconds to display in the visualization. This is a rolling range from the current time.",
					Validators: []validator.Int64{
						int64validator.ConflictsWith(
							path.MatchRoot("start_time"),
							path.MatchRoot("end_time"),
						),
						int64validator.AtLeast(0),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalUnitPrefix() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"unit_prefix": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Unit prefix for the chart.",
					Default:     stringdefault.StaticString("Metric"),
					Validators: []validator.String{
						stringvalidator.OneOf("Metric", "Binary"),
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithOptionalVizualisationOptions() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"viz_options": schema.ListNestedAttribute{
					Optional:    true,
					Description: "Visualization options for the chart.",
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"label": schema.StringAttribute{
								Required: true,
							},
							"color": schema.StringAttribute{
								Optional: true,
							},
							"display_name": schema.StringAttribute{
								Optional:    true,
								Description: "Display name for the visualization.",
							},
							"value_unit": schema.StringAttribute{
								Optional:    true,
								Description: "Unit for the value in the visualization.",
							},
							"value_prefix": schema.StringAttribute{
								Optional:    true,
								Description: "Prefix for the value in the visualization.",
							},
							"value_suffix": schema.StringAttribute{
								Optional:    true,
								Description: "Suffix for the value in the visualization.",
							},
						},
					},
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithRequiredMarkdown() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"markdown": schema.StringAttribute{
					Required:    true,
					Description: "Markdown content for the chart.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithRequiredSLOID() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"slo_id": schema.StringAttribute{
					Required:    true,
					Description: "The ID of the SLO associated with the chart.",
				},
			},
		}),
	}
}

func (sb *SchemaBuilder) WithRequiredProgramText() *SchemaBuilder {
	return &SchemaBuilder{
		applies: slices.Concat(sb.applies, []map[string]schema.Attribute{
			{
				"program_text": schema.StringAttribute{
					Required:    true,
					Description: "The Signalflow program text for the chart",
					MarkdownDescription: "Signalflow program text for the chart. " +
						"See [Signalflow documentation](https://help.splunk.com/en/splunk-observability-cloud/signalflow-analytics/signalflow-analytics) for more details.",
				},
			},
		}),
	}
}
