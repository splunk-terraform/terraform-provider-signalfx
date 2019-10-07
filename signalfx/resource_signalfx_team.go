package signalfx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	notification "github.com/signalfx/signalfx-go/notification"
	team "github.com/signalfx/signalfx-go/team"
)

const (
	TeamAppPath = "/detector/"
)

func teamResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the team",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the team (Optional)",
			},
			"members": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Members of team team",
			},
			"notifications_critical": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the critical alerts category.",
			},
			"notifications_default": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the default alerts category.",
			},
			"notifications_info": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the info alerts category.",
			},
			"notifications_major": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the major alerts category.",
			},
			"notifications_minor": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the minor alerts category.",
			},
			"notifications_warning": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the warning alerts category.",
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the team",
			},
		},

		Create: teamCreate,
		Read:   teamRead,
		Update: teamUpdate,
		Delete: teamDelete,
		Exists: teamExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create a team
*/
func getPayloadTeam(d *schema.ResourceData) (*team.CreateUpdateTeamRequest, error) {
	t := &team.CreateUpdateTeamRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	var members []string
	if val, ok := d.GetOk("values"); ok {
		tfValues := val.(*schema.Set).List()
		for _, v := range tfValues {
			members = append(members, v.(string))
		}
	}
	t.Members = members

	if val, ok := d.GetOk("notifications_critical"); ok {
		nots, err := getNotificationList(val.([]interface{}))
		if err != nil {
			return t, err
		}
		t.NotificationLists.Critical = nots
	}
	if val, ok := d.GetOk("notifications_default"); ok {
		nots, err := getNotificationList(val.([]interface{}))
		if err != nil {
			return t, err
		}
		t.NotificationLists.Default = nots
	}
	if val, ok := d.GetOk("notifications_info"); ok {
		nots, err := getNotificationList(val.([]interface{}))
		if err != nil {
			return t, err
		}
		t.NotificationLists.Info = nots
	}
	if val, ok := d.GetOk("notifications_major"); ok {
		nots, err := getNotificationList(val.([]interface{}))
		if err != nil {
			return t, err
		}
		t.NotificationLists.Major = nots
	}
	if val, ok := d.GetOk("notifications_minor"); ok {
		nots, err := getNotificationList(val.([]interface{}))
		if err != nil {
			return t, err
		}
		t.NotificationLists.Minor = nots
	}
	if val, ok := d.GetOk("notifications_warning"); ok {
		nots, err := getNotificationList(val.([]interface{}))
		if err != nil {
			return t, err
		}
		t.NotificationLists.Warning = nots
	}

	return t, nil
}

// Convert the list of TF data into proper objects
func getNotificationList(items []interface{}) ([]*notification.Notification, error) {
	if len(items) < 0 {
		return nil, nil
	}
	objects, err := getNotifications(items)
	if err != nil {
		return nil, err
	}

	return objects, nil
}

func getNotificationObject(item map[string]interface{}) (*notification.Notification, error) {
	t := item["type"].(string)
	var nValue interface{}
	switch t {
	case "BigPanda":
		nValue = &notification.BigPandaNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
		}
	case "Email":
		nValue = &notification.EmailNotification{
			Type:  t,
			Email: item["email"].(string),
		}
	case "Office365":
		nValue = &notification.Office365Notification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
		}
	case "Opsgenie":
		nValue = &notification.OpsgenieNotification{
			Type:          t,
			CredentialId:  item["credentialId"].(string),
			ResponderName: item["responderName"].(string),
			ResponderId:   item["responderId"].(string),
			ResponderType: item["responderType"].(string),
		}
	case "PagerDuty":
		nValue = &notification.PagerDutyNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
		}
	case "Slack":
		nValue = &notification.SlackNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
			Channel:      item["channel"].(string),
		}
	case "Team":
		nValue = &notification.TeamNotification{
			Type: t,
			Team: item["team"].(string),
		}
	case "TeamEmail":
		nValue = &notification.TeamEmailNotification{
			Type: t,
			Team: item["team"].(string),
		}
	case "VictorOps":
		nValue = &notification.VictorOpsNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
			RoutingKey:   item["routingKey"].(string),
		}
	case "Webhook":
		nValue = &notification.WebhookNotification{
			Type:   t,
			Secret: item["secret"].(string),
			Url:    item["url"].(string),
		}
	case "XMatters":
		nValue = &notification.XMattersNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
		}
	}

	return &notification.Notification{
		Type:  t,
		Value: nValue,
	}, nil
}

func getNotificationStringFromAPI(n *notification.Notification) (string, error) {
	switch n.Value.(type) {
	case *notification.BigPandaNotification:
		return fmt.Sprintf("%s,%s", n.Type, n.Value.(*notification.BigPandaNotification).CredentialId), nil
	case *notification.EmailNotification:
		return fmt.Sprintf("%s,%s", n.Type, n.Value.(*notification.EmailNotification).Email), nil
	case *notification.Office365Notification:
		return fmt.Sprintf("%s,%s", n.Type, n.Value.(*notification.Office365Notification).CredentialId), nil
	case *notification.OpsgenieNotification:
		ogn := n.Value.(*notification.OpsgenieNotification)
		return fmt.Sprintf("%s,%s,%s,%s,%s", n.Type, ogn.CredentialId, ogn.ResponderName, ogn.ResponderId, ogn.ResponderType), nil
	case *notification.PagerDutyNotification:
		return fmt.Sprintf("%s,%s", n.Type, n.Value.(*notification.PagerDutyNotification).CredentialId), nil
	case *notification.ServiceNowNotification:
		return fmt.Sprintf("%s,%s", n.Type, n.Value.(*notification.ServiceNowNotification).CredentialId), nil
	case *notification.SlackNotification:
		sn := n.Value.(*notification.SlackNotification)
		return fmt.Sprintf("%s,%s,%s", n.Type, sn.Channel, sn.CredentialId), nil
	case *notification.TeamNotification:
		tn := n.Value.(*notification.TeamNotification)
		return fmt.Sprintf("%s,%s", n.Type, tn.Team), nil
	case *notification.TeamEmailNotification:
		ten := n.Value.(*notification.TeamEmailNotification)
		return fmt.Sprintf("%s,%s", n.Type, ten.Team), nil
	case *notification.VictorOpsNotification:
		von := n.Value.(*notification.VictorOpsNotification)
		return fmt.Sprintf("%s,%s,%s", n.Type, von.CredentialId, von.RoutingKey), nil
	case *notification.WebhookNotification:
		whn := n.Value.(*notification.WebhookNotification)
		return fmt.Sprintf("%s,%s,%s", n.Type, whn.Secret, whn.Url), nil
	case *notification.XMattersNotification:
		return fmt.Sprintf("%s,%s", n.Type, n.Value.(*notification.XMattersNotification).CredentialId), nil
	default:
		return "", fmt.Errorf("Unknown notification type: %s", n.Type)
	}
}

func teamCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadTeam(d)
	if err != nil {
		return err
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Team Payload: %s", string(debugOutput))

	c, err := config.Client.CreateTeam(payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, TeamAppPath+c.Id)
	if err != nil {
		return err
	}
	d.Set("url", appURL)
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(c.Id)
	return teamAPIToTF(d, c)
}

func teamAPIToTF(d *schema.ResourceData, t *team.Team) error {
	debugOutput, _ := json.Marshal(t)
	log.Printf("[DEBUG] SignalFx: Got Team to enState: %s", string(debugOutput))

	if err := d.Set("name", t.Name); err != nil {
		return err
	}
	if err := d.Set("description", t.Description); err != nil {
		return err
	}

	if len(t.Members) > 0 {
		members := make([]interface{}, len(t.Members))
		for i, v := range t.Members {
			members[i] = v
		}
		if err := d.Set("members", schema.NewSet(schema.HashString, members)); err != nil {
			return err
		}
	}

	if len(t.NotificationLists.Critical) > 0 {
		nots, err := getNotificationsFromAPI(t.NotificationLists.Critical)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] SignalFx: CRITICAL %v", nots)
		d.Set("notifications_critical", nots)
	}
	if len(t.NotificationLists.Default) > 0 {
		nots, err := getNotificationsFromAPI(t.NotificationLists.Default)
		if err != nil {
			return err
		}
		d.Set("notifications_default", nots)
	}
	if len(t.NotificationLists.Info) > 0 {
		nots, err := getNotificationsFromAPI(t.NotificationLists.Info)
		if err != nil {
			return err
		}
		d.Set("notifications_info", nots)
	}
	if len(t.NotificationLists.Major) > 0 {
		nots, err := getNotificationsFromAPI(t.NotificationLists.Major)
		if err != nil {
			return err
		}
		d.Set("notifications_major", nots)
	}
	if len(t.NotificationLists.Minor) > 0 {
		nots, err := getNotificationsFromAPI(t.NotificationLists.Minor)
		if err != nil {
			return err
		}
		d.Set("notifications_minor", nots)
	}
	if len(t.NotificationLists.Warning) > 0 {
		nots, err := getNotificationsFromAPI(t.NotificationLists.Warning)
		if err != nil {
			return err
		}
		d.Set("notifications_warning", nots)
	}
	return nil
}

func getNotificationsFromAPI(nots []*notification.Notification) ([]string, error) {
	results := make([]string, len(nots))
	for i, not := range nots {
		s, err := getNotificationStringFromAPI(not)
		if err != nil {
			return nil, err
		}
		results[i] = s
	}
	return results, nil
}

func teamRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	c, err := config.Client.GetTeam(d.Id())
	if err != nil {
		return err
	}

	return teamAPIToTF(d, c)
}

func teamUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadTeam(d)
	if err != nil {
		return err
	}
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Team Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateTeam(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Team Response: %v", c)

	d.SetId(c.Id)
	return teamAPIToTF(d, c)
}

func teamDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteTeam(d.Id())
}

func teamExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetTeam(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "Bad status 404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
