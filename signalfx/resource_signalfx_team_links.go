package signalfx

import (
	"context"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func teamLinksResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"team": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Team ID to link the configured dashboard groups or detectos to.",
			},
			"dashboard_groups": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Dashboard group IDs to link to the team.",
				Optional:    true,
			},
			"detectors": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Detector IDs to link to the team.",
				Optional:    true,
			},
		},
		Create: teamLinksCreate,
		Read:   teamLinksRead,
		Update: teamLinksUpdate,
		Delete: teamLinksDelete,
		Exists: teamLinksExists,
	}
}

func teamLinksExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	return true, nil
}

func teamLinksRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	teamId := d.Get("team").(string)
	res, err := config.Client.GetTeam(context.TODO(), teamId)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	_, new := d.GetChange("detectors")
	detectors := new.(*schema.Set)

	dets := make([]interface{}, len(res.Detectors))
	for i, detector := range res.Detectors {
		dets[i] = detector
	}

	teamDets := schema.NewSet(schema.HashString, dets)
	// XXX: For some reason the same sets are hashing to different values (maybe to interface{} vs string types).
	// Rehash the set so intersection, etc. will work.
	detectors = schema.NewSet(schema.HashString, detectors.List())

	if err := d.Set("detectors", teamDets.Intersection(detectors)); err != nil {
		return err
	}

	_, new = d.GetChange("dashboard_groups")
	dashboardGroups := new.(*schema.Set)

	dbgs := make([]interface{}, len(res.DashboardGroups))
	for i, dbg := range res.DashboardGroups {
		dbgs[i] = dbg
	}

	teamDbg := schema.NewSet(schema.HashString, dbgs)
	// XXX: Rehash for same reason above.
	dashboardGroups = schema.NewSet(schema.HashString, dashboardGroups.List())

	if err := d.Set("dashboard_groups", teamDbg.Intersection(dashboardGroups)); err != nil {
		return err
	}

	return nil
}

func teamLinksCreate(d *schema.ResourceData, meta interface{}) error {
	return teamLinksUpdate(d, meta)
}

func teamLinksUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	team := d.Get("team").(string)

	old, new := d.GetChange("detectors")
	oldD := old.(*schema.Set)
	newD := new.(*schema.Set)

	removed := oldD.Difference(newD)
	added := newD.Difference(oldD)

	log.Printf("[DEBUG] SignalFx: Unlinking detectors from team %v: %v", team, removed.List())

	for _, rd := range removed.List() {
		det := rd.(string)
		if err := config.Client.UnlinkDetectorFromTeam(context.TODO(), team, det); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] SignalFx: Linking detectors to team %v: %v", team, added.List())

	for _, d := range added.List() {
		det := d.(string)
		if err := config.Client.LinkDetectorToTeam(context.TODO(), team, det); err != nil {
			return err
		}
	}

	if err := d.Set("detectors", new); err != nil {
		return err
	}

	old, new = d.GetChange("dashboard_groups")
	oldDbg := old.(*schema.Set)
	newDbg := new.(*schema.Set)

	removed = oldDbg.Difference(newDbg)
	added = newDbg.Difference(oldDbg)

	log.Printf("[DEBUG] SignalFx: Unlinking dashboard groups from team %v: %v", team, removed.List())

	for _, d := range removed.List() {
		det := d.(string)
		if err := config.Client.UnlinkDashboardGroupFromTeam(context.TODO(), team, det); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] SignalFx: Linking dashboard groups to team %v: %v", team, added.List())

	for _, d := range added.List() {
		det := d.(string)
		if err := config.Client.LinkDashboardGroupToTeam(context.TODO(), team, det); err != nil {
			return err
		}
	}

	if err := d.Set("dashboard_groups", new); err != nil {
		return err
	}

	d.SetId("0")

	return nil
}

func teamLinksDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	team := d.Get("team").(string)
	detectors := d.Get("detectors").(*schema.Set)
	dashboardGroups := d.Get("dashboard_groups").(*schema.Set)

	for _, d := range dashboardGroups.List() {
		dbg := d.(string)
		if err := config.Client.UnlinkDashboardGroupFromTeam(context.TODO(), team, dbg); err != nil {
			return err
		}
	}

	for _, d := range detectors.List() {
		det := d.(string)
		if err := config.Client.UnlinkDetectorFromTeam(context.TODO(), team, det); err != nil {
			return err
		}
	}

	return nil
}
