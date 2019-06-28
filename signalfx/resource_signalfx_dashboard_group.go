package signalfx

import (
	"encoding/json"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	dashboard_group "github.com/signalfx/signalfx-go/dashboard_group"
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
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Team IDs to associate the dashboard group to",
			},
		},

		Create: dashboardgroupCreate,
		Read:   dashboardgroupRead,
		Update: dashboardgroupUpdate,
		Delete: dashboardgroupDelete,
	}
}

/*
  Use Resource object to construct json payload in order to create a dasboard group
*/
func getPayloadDashboardGroup(d *schema.ResourceData) *dashboard_group.CreateUpdateDashboardGroupRequest {
	cudgr := &dashboard_group.CreateUpdateDashboardGroupRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	if val, ok := d.GetOk("teams"); ok {
		teams := []string{}
		for _, t := range val.([]interface{}) {
			teams = append(teams, t.(string))
		}
		cudgr.Teams = teams
	}

	return cudgr
}

func dashboardgroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadDashboardGroup(d)

	dg, err := config.Client.CreateDashboardGroup(payload)
	if err != nil {
		return err
	}
	d.SetId(dg.Id)

	return dashboardGroupAPIToTF(d, dg)
}

func dashboardGroupAPIToTF(d *schema.ResourceData, dg *dashboard_group.DashboardGroup) error {
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

	return nil
}

func dashboardgroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	dg, err := config.Client.GetDashboardGroup(d.Id())
	if err != nil {
		return err
	}

	return dashboardGroupAPIToTF(d, dg)
}

func dashboardgroupUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadDashboardGroup(d)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Dashboard Group Payload: %s", string(debugOutput))

	dg, err := config.Client.UpdateDashboardGroup(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Dashboard Group Response: %v", dg)

	d.SetId(dg.Id)
	return dashboardGroupAPIToTF(d, dg)
}

func dashboardgroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteDashboardGroup(d.Id())
}
