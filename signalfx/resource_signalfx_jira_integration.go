package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/signalfx/signalfx-go/integration"
)

var validAuthMethod = regexp.MustCompile("^(UsernameAndPassword|EmailAndToken)$")

func integrationJiraResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the integration",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the integration is enabled or not",
			},
			"api_token": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"username", "password"},
				Description:   "The API token for the user email",
			},
			"user_email": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"username", "password"},
				Description:   "Email address used to authenticate the Jira integration.",
			},
			"username": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"user_email", "api_token"},
				Description:   "User name used to authenticate the Jira integration.",
			},
			"password": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"user_email", "api_token"},
				Description:   "Password used to authenticate the Jira integration.",
			},
			"auth_method": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(validAuthMethod, "must be one of `UsernameAndPassword` or `EmailAndToken`"),
				Description:  "Authentication method used when creating the Jira integration. One of `EmailAndToken` or `UsernameAndPassword`",
			},
			"base_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Base URL of the Jira instance that's integrated with SignalFx.",
			},
			"issue_type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Issue type (for example, Story) for tickets that Jira creates for detector notifications. SignalFx validates issue types, so you must specify a type that's valid for the Jira project specified in `projectKey`.",
			},
			"project_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Jira key of an existing project. When Jira creates a new ticket for a detector notification, the ticket is assigned to this project.",
			},
			"assignee_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Jira user name for the assignee",
			},
			"assignee_display_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Jira display name for the assignee",
			},
		},

		Create: integrationJiraCreate,
		Read:   integrationJiraRead,
		Update: integrationJiraUpdate,
		Delete: integrationJiraDelete,
		Exists: integrationJiraExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationJiraExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetJiraIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func integrationJiraRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetJiraIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return jiraIntegrationAPIToTF(d, int)
}

func jiraIntegrationAPIToTF(d *schema.ResourceData, jira *integration.JiraIntegration) error {
	debugOutput, _ := json.Marshal(jira)
	log.Printf("[DEBUG] SignalFx: Got Jira Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", jira.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", jira.Enabled); err != nil {
		return err
	}
	if err := d.Set("auth_method", jira.AuthMethod); err != nil {
		return err
	}
	if err := d.Set("base_url", jira.BaseURL); err != nil {
		return err
	}
	if err := d.Set("issue_type", jira.IssueType); err != nil {
		return err
	}
	if err := d.Set("project_key", jira.ProjectKey); err != nil {
		return err
	}
	// Only gonna do one of these, depending in the value
	if jira.AuthMethod == "UsernameAndPassword" {
		if err := d.Set("username", jira.Username); err != nil {
			return err
		}
		// We don't set password because it isn't returned.
	} else {
		// Ok, then it's the email version
		if err := d.Set("user_email", jira.UserEmail); err != nil {
			return err
		}
		// We don't set api_token because it isn't returned.
	}

	if err := d.Set("assignee_name", jira.Assignee.Name); err != nil {
		return err
	}
	if jira.Assignee.DisplayName != "" {
		if err := d.Set("assignee_display_name", jira.Assignee.DisplayName); err != nil {
			return err
		}
	}

	return nil
}

func getPayloadJiraIntegration(d *schema.ResourceData) (*integration.JiraIntegration, error) {

	jira := &integration.JiraIntegration{
		Name:       d.Get("name").(string),
		Type:       "Jira",
		Enabled:    d.Get("enabled").(bool),
		AuthMethod: d.Get("auth_method").(string),
		BaseURL:    d.Get("base_url").(string),
		IssueType:  d.Get("issue_type").(string),
		ProjectKey: d.Get("project_key").(string),
		Assignee: &integration.JiraAssignee{
			Name: d.Get("assignee_name").(string),
		},
	}

	if val, ok := d.GetOk("assignee_display_name"); ok {
		jira.Assignee.DisplayName = val.(string)
	}

	if jira.AuthMethod == "UsernameAndPassword" {
		jira.Username = d.Get("username").(string)
		jira.Password = d.Get("password").(string)
	} else {
		jira.UserEmail = d.Get("user_email").(string)
		jira.APIToken = d.Get("api_token").(string)
	}

	return jira, nil
}

func integrationJiraCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadJiraIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Jira Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateJiraIntegration(context.TODO(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return jiraIntegrationAPIToTF(d, int)
}

func integrationJiraUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadJiraIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Jira Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateJiraIntegration(context.TODO(), d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return jiraIntegrationAPIToTF(d, int)
}

func integrationJiraDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteJiraIntegration(context.TODO(), d.Id())
}
