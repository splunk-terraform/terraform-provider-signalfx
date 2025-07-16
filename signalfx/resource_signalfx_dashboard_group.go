// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	dashboard_group "github.com/signalfx/signalfx-go/dashboard_group"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func dashboardGroupResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the dashboard group",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the dashboard group",
			},
			"teams": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Team IDs to associate the dashboard group to",
			},
			"dashboard": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Dashboard IDs that are members of this dashboard group. Also handles 'mirrored' dashboards.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"config_id": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Unique identifier of an association between a dashboard group and a dashboard",
						},
						"dashboard_id": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The label used in the publish statement that displays the plot (metric time series data) you want to customize",
						},
						"description_override": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "String that provides a description override for a mirrored dashboard",
						},
						"name_override": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "String that provides a name override for a mirrored dashboard",
						},
						"filter_override": &schema.Schema{
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "Filter to apply to each chart in the dashboard",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"property": &schema.Schema{
										Type:        schema.TypeString,
										Required:    true,
										Description: "A metric time series dimension or property name",
									},
									"values": &schema.Schema{
										Type:        schema.TypeSet,
										Required:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Description: "List of strings (which will be treated as an OR filter on the property)",
									},
									"negated": &schema.Schema{
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "(false by default) Whether this filter should be a \"not\" filter",
									},
								},
							},
						},
						"variable_override": &schema.Schema{
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "Dashboard variable to apply to each chart in the dashboard",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"property": &schema.Schema{
										Type:        schema.TypeString,
										Required:    true,
										Description: "A metric time series dimension or property name",
									},
									"values": &schema.Schema{
										Type:        schema.TypeSet,
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Description: "List of strings (which will be treated as an OR filter on the property)",
									},
									"values_suggested": &schema.Schema{
										Type:        schema.TypeSet,
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Description: "A list of strings of suggested values for this variable; these suggestions will receive priority when values are autosuggested for this variable",
									},
								},
							},
						},
					},
				},
			},
			"authorized_writer_teams": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Deprecated:  "Please use permissions field now",
				Description: "Team IDs that have write access to this dashboard",
			},
			"authorized_writer_users": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Deprecated:  "Please use permissions field now",
				Description: "User IDs that have write access to this dashboard",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Description: "The custom access control list for this dashboard",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"principal_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the principal with access",
						},
						"principal_type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"ORG", "TEAM", "USER"}, false),
							Description:  "Type of principal, possible values: ORG, TEAM, USER",
						},
						"actions": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{"READ", "WRITE"}, false),
							},
							Description: "Actions level, possible values: READ, WRITE",
						},
					},
				},
			},
			"import_qualifier": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metric": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"filters": &schema.Schema{
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "Filter to apply to each chart in the dashboard",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"property": &schema.Schema{
										Type:        schema.TypeString,
										Required:    true,
										Description: "A metric time series dimension or property name",
									},
									"values": &schema.Schema{
										Type:        schema.TypeSet,
										Required:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Description: "List of strings (which will be treated as an OR filter on the property)",
									},
									"negated": &schema.Schema{
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "(false by default) Whether this filter should be a \"not\" filter",
									},
								},
							},
						},
					},
				},
			},
		},

		Create: dashboardgroupCreate,
		Read:   dashboardgroupRead,
		Update: dashboardgroupUpdate,
		Delete: dashboardgroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
Use Resource object to construct json payload in order to create a dasboard group
*/
func getPayloadDashboardGroup(d *schema.ResourceData) *dashboard_group.CreateUpdateDashboardGroupRequest {
	cudgr := &dashboard_group.CreateUpdateDashboardGroupRequest{
		Name:              d.Get("name").(string),
		Description:       d.Get("description").(string),
		AuthorizedWriters: &dashboard_group.AuthorizedWriters{},
	}

	if val, ok := d.GetOk("teams"); ok {
		teams := []string{}
		for _, t := range val.(*schema.Set).List() {
			teams = append(teams, t.(string))
		}
		cudgr.Teams = teams
	}

	if val, ok := d.GetOk("authorized_writer_teams"); ok {
		var teams []string
		tfValues := val.(*schema.Set).List()
		for _, v := range tfValues {
			teams = append(teams, v.(string))
		}
		cudgr.AuthorizedWriters.Teams = teams
	}
	if val, ok := d.GetOk("authorized_writer_users"); ok {
		var users []string
		tfValues := val.(*schema.Set).List()
		for _, v := range tfValues {
			users = append(users, v.(string))
		}
		cudgr.AuthorizedWriters.Users = users
	}
	permissions := getPermissions(d)
	if permissions.Acl != nil {
		cudgr.Permissions = permissions
	}

	// Because at present the `DashboardConfigs` mirrors the `Dashboards`
	// field, we need to pay close attention here. We should only treat
	// this as a mirror if one of the configs has one of the mirrored fields set.
	hasMirrors := false

	if dashes, ok := d.GetOk("dashboard"); ok {
		dashboards := dashes.([]interface{})
		dashConfigs := make([]*dashboard_group.DashboardConfig, len(dashboards))
		for i, d := range dashboards {
			dash := d.(map[string]interface{})

			dcon := &dashboard_group.DashboardConfig{
				DashboardId: dash["dashboard_id"].(string),
			}

			if descOver, ok := dash["description_override"]; ok && descOver != "" {
				dcon.DescriptionOverride = descOver.(string)
				hasMirrors = true
			}
			if nameOver, ok := dash["name_override"]; ok && nameOver != "" {
				dcon.NameOverride = nameOver.(string)
				hasMirrors = true
			}

			filtersOverride := &dashboard_group.Filters{}

			if filterOver, ok := dash["filter_override"]; ok {
				hasMirrors = true
				filterOver := filterOver.(*schema.Set).List()
				filters := make([]*dashboard_group.Filter, len(filterOver))
				for i, f := range filterOver {
					f := f.(map[string]interface{})
					var values []string
					tfValues := f["values"].(*schema.Set).List()
					if len(tfValues) > 0 {
						values = []string{}
						for _, v := range tfValues {
							values = append(values, v.(string))
						}
					}

					filters[i] = &dashboard_group.Filter{
						NOT:      f["negated"].(bool),
						Property: f["property"].(string),
						Values:   values,
					}
				}

				filtersOverride.Sources = filters
			}

			if variableOver, ok := dash["variable_override"]; ok {
				hasMirrors = true
				tfVars := variableOver.(*schema.Set).List()
				vars := make([]*dashboard_group.WebUiFilter, len(tfVars))
				for i, v := range tfVars {
					v := v.(map[string]interface{})
					var values []string
					tfValues := v["values"].(*schema.Set).List()
					if len(tfValues) > 0 {
						values = []string{}
						for _, v := range tfValues {
							values = append(values, v.(string))
						}
					}

					var preferredSuggestions []string
					if val, ok := v["values_suggested"]; ok {
						tfValues := val.(*schema.Set).List()
						for _, v := range tfValues {
							preferredSuggestions = append(preferredSuggestions, v.(string))
						}
					}

					vars[i] = &dashboard_group.WebUiFilter{
						PreferredSuggestions: preferredSuggestions,
						Property:             v["property"].(string),
						Value:                values,
					}
				}
				filtersOverride.Variables = vars
			}

			if len(filtersOverride.Sources) > 0 || len(filtersOverride.Variables) > 0 {
				dcon.FiltersOverride = filtersOverride
			}

			dashConfigs[i] = dcon
		}
		if hasMirrors {
			log.Println("[DEBUG] SignalFx: We have mirrors, adding them")
			cudgr.DashboardConfigs = dashConfigs
		}
	}

	if tfiq, ok := d.GetOk("import_qualifier"); ok {
		tfIQs := tfiq.(*schema.Set).List()
		iqs := make([]*dashboard_group.ImportQualifier, len(tfIQs))
		for i, iq := range tfIQs {
			iq := iq.(map[string]interface{})

			filterOver := iq["filters"].(*schema.Set).List()
			filters := make([]*dashboard_group.ImportFilter, len(filterOver))
			for i, f := range filterOver {
				f := f.(map[string]interface{})
				var values []string
				tfValues := f["values"].(*schema.Set).List()
				if len(tfValues) > 0 {
					values = []string{}
					for _, v := range tfValues {
						values = append(values, v.(string))
					}
				}

				filters[i] = &dashboard_group.ImportFilter{
					NOT:      f["negated"].(bool),
					Property: f["property"].(string),
					Values:   values,
				}
			}

			iqs[i] = &dashboard_group.ImportQualifier{
				Metric:  iq["metric"].(string),
				Filters: filters,
			}
		}
		cudgr.ImportQualifiers = iqs
	}

	return cudgr
}

func getPermissions(d *schema.ResourceData) *dashboard_group.ObjectPermissions {
	permissions := &dashboard_group.ObjectPermissions{}
	if val := getPermissionsAcl(d); len(val) > 0 {
		permissions.Acl = val
	}
	return permissions
}

func getPermissionsAcl(d *schema.ResourceData) []*dashboard_group.AclEntry {
	acl := d.Get("permissions").(*schema.Set).List()
	aclList := make([]*dashboard_group.AclEntry, len(acl))
	for i, entry := range acl {
		entry := entry.(map[string]interface{})

		item := &dashboard_group.AclEntry{
			PrincipalId:   entry["principal_id"].(string),
			PrincipalType: entry["principal_type"].(string),
			Actions:       convert.SchemaListAll(entry["actions"], convert.ToString),
		}
		aclList[i] = item
	}
	return aclList
}

func dashboardgroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadDashboardGroup(d)

	payload.Teams = pmeta.MergeProviderTeams(context.TODO(), meta, payload.Teams)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Dashboard Group Create Payload: %s", debugOutput)

	dg, err := config.Client.CreateDashboardGroup(context.TODO(), payload, true)
	if err != nil {
		return err
	}
	d.SetId(dg.Id)

	return dashboardGroupAPIToTF(d, dg, meta)
}

func dashboardGroupAPIToTF(d *schema.ResourceData, dg *dashboard_group.DashboardGroup, meta interface{}) error {
	debugOutput, _ := json.Marshal(dg)
	log.Printf("[DEBUG] SignalFx: Got Dashboard Group to enState: %s", string(debugOutput))

	if err := d.Set("name", dg.Name); err != nil {
		return err
	}
	if err := d.Set("description", dg.Description); err != nil {
		return err
	}
	if err := d.Set("teams", dg.Teams); err != nil {
		return err
	}

	if dg.AuthorizedWriters != nil {
		aw := dg.AuthorizedWriters
		if aw.Teams != nil && len(aw.Teams) > 0 {
			teams := make([]interface{}, len(aw.Teams))
			for i, v := range aw.Teams {
				teams[i] = v
			}
			if err := d.Set("authorized_writer_teams", schema.NewSet(schema.HashString, teams)); err != nil {
				return err
			}
		}
		if aw.Users != nil && len(aw.Users) > 0 {
			users := make([]interface{}, len(aw.Users))
			for i, v := range aw.Users {
				users[i] = v
			}
			if err := d.Set("authorized_writer_users", schema.NewSet(schema.HashString, users)); err != nil {
				return err
			}
		}
	}

	if dg.Permissions != nil {
		p := dg.Permissions
		if p.Acl != nil && len(p.Acl) > 0 {
			acl := make([]map[string]interface{}, len(p.Acl))
			for i, a := range p.Acl {
				entry := make(map[string]interface{})
				entry["principal_id"] = a.PrincipalId
				entry["principal_type"] = a.PrincipalType
				entry["actions"] = flattenStringSliceToSet(a.Actions)
				acl[i] = entry
			}
			if err := d.Set("permissions", acl); err != nil {
				return err
			}
		}
	}

	if len(dg.DashboardConfigs) > 0 {
		// Collect a list of mirrored dashboard configs
		config := meta.(*signalfxConfig)
		mirroredDashboardConfigs, err := getMirroredDashboardConfigs(config, d)
		if err != nil {
			return err
		}

		dConfigs := make([]map[string]interface{}, len(mirroredDashboardConfigs))
		for i, dc := range mirroredDashboardConfigs {
			dConf := make(map[string]interface{})
			dConf["config_id"] = dc.ConfigId
			dConf["dashboard_id"] = dc.DashboardId
			dConf["description_override"] = dc.DescriptionOverride
			dConf["name_override"] = dc.NameOverride

			if dc.FiltersOverride != nil {
				if len(dc.FiltersOverride.Sources) > 0 {
					sources := make([]map[string]interface{}, len(dc.FiltersOverride.Sources))
					for i, s := range dc.FiltersOverride.Sources {
						source := make(map[string]interface{})
						source["negated"] = s.NOT
						source["property"] = s.Property
						source["values"] = flattenStringSliceToSet(s.Values)
						sources[i] = source
					}
					dConf["filter_override"] = sources
				}

				if len(dc.FiltersOverride.Variables) > 0 {
					vars := make([]map[string]interface{}, len(dc.FiltersOverride.Variables))
					for i, v := range dc.FiltersOverride.Variables {
						dvar := make(map[string]interface{})
						dvar["property"] = v.Property
						/* API Seems to require alias, but doesn't seem to end up using the value at all
							 Pass it here to avoid API issues
						*/
						dvar["alias"] = v.Property
						dvar["values"] = flattenStringSliceToSet(v.Value)
						dvar["values_suggested"] = flattenStringSliceToSet(v.PreferredSuggestions)
						vars[i] = dvar
					}
					dConf["variable_override"] = vars
				}
			}

			dConfigs[i] = dConf
		}
		if len(mirroredDashboardConfigs) > 0 {
			if err := d.Set("dashboard", dConfigs); err != nil {
				return err
			}
		}
	}

	if len(dg.ImportQualifiers) > 0 {
		iqs := make([]map[string]interface{}, len(dg.ImportQualifiers))
		for i, apiIQ := range dg.ImportQualifiers {
			iq := make(map[string]interface{})
			iq["metric"] = apiIQ.Metric
			filters := make([]map[string]interface{}, len(apiIQ.Filters))
			for j, apiFilter := range apiIQ.Filters {
				filter := make(map[string]interface{})
				filter["negated"] = apiFilter.NOT
				filter["property"] = apiFilter.Property
				filter["values"] = flattenStringSliceToSet(apiFilter.Values)
				filters[j] = filter
			}
			iq["filters"] = filters
			iqs[i] = iq
		}
		if err := d.Set("import_qualifier", iqs); err != nil {
			return err
		}
	}

	return nil
}

func dashboardgroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	dg, err := config.Client.GetDashboardGroup(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	return dashboardGroupAPIToTF(d, dg, meta)
}

func dashboardgroupUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadDashboardGroup(d)

	// The Terraform provider does not track non-mirrored dashboards in dashboard groups.
	// It keeps track dashboardgroup membership on the respective dashboards. However,
	// the backend modifies dashboard groups object to hold a list of all dashboards
	// it holds, both mirrored and non-mirrored. The API expects this full list of
	// dashboards whenever mirrored dashboards are added/present. Collect all
	// non-mirrored dashboards from the backend and append it to the list of dashboards.
	// This behavior is noted in step 4 of the API docs here:
	// https://dev.splunk.com/observability/docs/chartsdashboards/dashboard_groups_overview#Add-the-mirrored-dashboard
	nonMirroredDashes, err := getNonMirroredDashes(config, d)
	if err != nil {
		return fmt.Errorf("failed to get current dashboard list for %s: %v", d.Id(), err)
	}
	payload.Teams = pmeta.MergeProviderTeams(context.TODO(), meta, payload.Teams)

	payload.DashboardConfigs = append(payload.DashboardConfigs, nonMirroredDashes...)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Dashboard Group Payload: %s", string(debugOutput))

	dg, err := config.Client.UpdateDashboardGroup(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] SignalFx: Update Dashboard Group Response: %v", dg)

	d.SetId(dg.Id)
	return dashboardGroupAPIToTF(d, dg, meta)
}

func getNonMirroredDashes(config *signalfxConfig, d *schema.ResourceData) ([]*dashboard_group.DashboardConfig, error) {
	mirrorIDsToBeOmitted := map[string]bool{}
	mirroredDashboardConfigs, err := getMirroredDashboardConfigs(config, d)
	if err != nil {
		return nil, fmt.Errorf("failed to get mirrored dashboard list for %s: %v", d.Id(), err)
	} else {
		for _, dc := range mirroredDashboardConfigs {
			mirrorIDsToBeOmitted[dc.DashboardId] = true
		}
	}

	dg, err := config.Client.GetDashboardGroup(context.TODO(), d.Id())
	if err != nil {
		return nil, err
	}

	out := make([]*dashboard_group.DashboardConfig, 0, len(dg.DashboardConfigs))
	for _, dc := range dg.DashboardConfigs {
		if !mirrorIDsToBeOmitted[dc.DashboardId] {
			out = append(out, &dashboard_group.DashboardConfig{DashboardId: dc.DashboardId, ConfigId: dc.ConfigId})
		}
	}

	return out, nil
}

func getMirroredDashboardConfigs(config *signalfxConfig, d *schema.ResourceData) ([]*dashboard_group.DashboardConfig, error) {
	dg, err := config.Client.GetDashboardGroup(context.TODO(), d.Id())
	if err != nil {
		return nil, err
	}

	out := make([]*dashboard_group.DashboardConfig, 0, len(dg.DashboardConfigs))
	for _, dc := range dg.DashboardConfigs {
		dash, err := config.Client.GetDashboard(context.TODO(), dc.DashboardId)
		if err != nil || dash.GroupId == d.Id() {
			// It is not a mirrored dashboard
			// if the dashboard's group ID matches with the current dashboard group id
			continue
		}
		out = append(out, dc)
	}

	return out, nil
}

func dashboardgroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteDashboardGroup(context.TODO(), d.Id())
}
