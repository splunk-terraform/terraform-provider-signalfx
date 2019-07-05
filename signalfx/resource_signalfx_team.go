package signalfx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	team "github.com/signalfx/signalfx-go/team"
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
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Members of team team",
			},
			"notifications_critical": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the critical alerts category.",
			},
			"notifications_default": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the default alerts category.",
			},
			"notifications_info": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the info alerts category.",
			},
			"notifications_major": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the major alerts category.",
			},
			"notifications_minor": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of notification destinations to use for the minor alerts category.",
			},
			"notifications_warning": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
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
  Use Resource object to construct json payload in order to create a text chart
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

	if val, ok := d.GetOk("notifications_default"); ok {
		defaults, err := getNotificationList(val.(*schema.Set).List())
		if err != nil {
			return t, err
		}
		t.NotificationLists.Default = defaults
	}

	return t, nil
}

// Convert the list of TF data into proper objects
func getNotificationList(items []interface{}) ([]*team.Notification, error) {
	if len(items) < 0 {
		return nil, nil
	}
	objects := getNotifications(items)
	notifs := make([]*team.Notification, len(objects))
	for i, item := range objects {
		notif, err := getNotificationObject(item)
		if err != nil {
			return nil, err
		}
		notifs[i] = notif
	}

	return notifs, nil
}

func getNotificationObject(item map[string]interface{}) (*team.Notification, error) {
	t := item["type"].(string)
	var nValue interface{}
	switch t {
	case "BigPanda":
		nValue = &team.BigPandaNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
		}
	case "Email":
		nValue = &team.EmailNotification{
			Type:  t,
			Email: item["email"].(string),
		}
	case "Office365":
		nValue = &team.Office365Notification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
		}
	case "Opsgenie":
		nValue = &team.OpsgenieNotification{
			Type:           t,
			CredentialId:   item["credentialId"].(string),
			CredentialName: item["credentialName"].(string),
			ResponderName:  item["responderName"].(string),
			ResponderId:    item["responderId"].(string),
			ResponderType:  item["responderType"].(string),
		}
	case "PagerDuty":
		nValue = &team.PagerDutyNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
		}
	case "Slack":
		nValue = &team.SlackNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
			Channel:      item["channel"].(string),
		}
	case "Team":
		nValue = &team.TeamNotification{
			Type: t,
			Team: item["team"].(string),
		}
	case "TeamEmail":
		nValue = &team.TeamEmailNotification{
			Type: t,
			Team: item["team"].(string),
		}
	case "VictorOps":
		nValue = &team.VictorOpsNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
			RoutingKey:   item["routingKey"].(string),
		}
	case "Webhook":
		nValue = &team.WebhookNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
			Secret:       item["secret"].(string),
			Url:          item["url"].(string),
		}
	case "XMatters":
		nValue = &team.XMattersNotification{
			Type:         t,
			CredentialId: item["credentialId"].(string),
		}
	}

	return &team.Notification{
		Type:  t,
		Value: nValue,
	}, nil
}

func getNotificationStringFromAPI(notification *team.Notification) (string, error) {
	switch notification.Value.(type) {
	case *team.BigPandaNotification:
		return fmt.Sprintf("%s,%s", notification.Type, notification.Value.(*team.BigPandaNotification).CredentialId), nil
	case *team.EmailNotification:
		return fmt.Sprintf("%s,%s", notification.Type, notification.Value.(*team.EmailNotification).Email), nil
	case *team.Office365Notification:
		return fmt.Sprintf("%s,%s", notification.Type, notification.Value.(*team.Office365Notification).CredentialId), nil
	case *team.OpsgenieNotification:
		ogn := notification.Value.(*team.OpsgenieNotification)
		return fmt.Sprintf("%s,%s,%s,%s,%s,%s", notification.Type, ogn.CredentialId, ogn.CredentialName, ogn.ResponderName, ogn.ResponderId, ogn.ResponderType), nil
	case *team.PagerDutyNotification:
		return fmt.Sprintf("%s,%s", notification.Type, notification.Value.(*team.PagerDutyNotification).CredentialId), nil
	case *team.ServiceNowNotification:
		return fmt.Sprintf("%s,%s", notification.Type, notification.Value.(*team.ServiceNowNotification).CredentialId), nil
	case *team.SlackNotification:
		sn := notification.Value.(*team.SlackNotification)
		return fmt.Sprintf("%s,%s,%s", notification.Type, sn.Channel, sn.CredentialId), nil
	case *team.TeamNotification:
		tn := notification.Value.(*team.TeamNotification)
		return fmt.Sprintf("%s,%s", notification.Type, tn.Team), nil
	case *team.TeamEmailNotification:
		ten := notification.Value.(*team.TeamEmailNotification)
		return fmt.Sprintf("%s,%s", notification.Type, ten.Team), nil
	case *team.VictorOpsNotification:
		von := notification.Value.(*team.VictorOpsNotification)
		return fmt.Sprintf("%s,%s,%s", notification.Type, von.CredentialId, von.RoutingKey), nil
	case *team.WebhookNotification:
		whn := notification.Value.(*team.WebhookNotification)
		return fmt.Sprintf("%s,%s,%s,%s", notification.Type, whn.CredentialId, whn.Secret, whn.Url), nil
	case *team.XMattersNotification:
		return fmt.Sprintf("%s,%s", notification.Type, notification.Value.(*team.XMattersNotification).CredentialId), nil
	default:
		return "", fmt.Errorf("Unknown notification type: %s", notification.Type)
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
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
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
		nots, err := getNotificationSetFromAPI(t.NotificationLists.Critical)
		if err != nil {
			return err
		}
		d.Set("notifications_critical", nots)
	}
	if len(t.NotificationLists.Default) > 0 {
		nots, err := getNotificationSetFromAPI(t.NotificationLists.Default)
		if err != nil {
			return err
		}
		d.Set("notifications_default", nots)
	}
	if len(t.NotificationLists.Info) > 0 {
		nots, err := getNotificationSetFromAPI(t.NotificationLists.Info)
		if err != nil {
			return err
		}
		d.Set("notifications_info", nots)
	}
	if len(t.NotificationLists.Major) > 0 {
		nots, err := getNotificationSetFromAPI(t.NotificationLists.Major)
		if err != nil {
			return err
		}
		d.Set("notifications_major", nots)
	}
	if len(t.NotificationLists.Minor) > 0 {
		nots, err := getNotificationSetFromAPI(t.NotificationLists.Minor)
		if err != nil {
			return err
		}
		d.Set("notifications_minor", nots)
	}
	if len(t.NotificationLists.Warning) > 0 {
		nots, err := getNotificationSetFromAPI(t.NotificationLists.Warning)
		if err != nil {
			return err
		}
		d.Set("notifications_warning", nots)
	}

	return nil
}

func getNotificationSetFromAPI(nots []*team.Notification) (*schema.Set, error) {
	results := make([]interface{}, len(nots))
	for i, not := range nots {
		s, err := getNotificationStringFromAPI(not)
		if err != nil {
			return nil, err
		}
		results[i] = s
	}
	return schema.NewSet(schema.HashString, results), nil
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
