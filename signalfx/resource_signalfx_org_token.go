package signalfx

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/signalfx/signalfx-go/orgtoken"
)

func orgTokenResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the token",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the token (Optional)",
			},
			"disabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag that controls enabling the token. If set to `true`, the token is disabled, and you can't use it for authentication. Defaults to `false`",
			},
		},

		Create: orgTokenCreate,
		Read:   orgTokenRead,
		Update: orgTokenUpdate,
		Delete: orgTokenDelete,
		Exists: orgTokenExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func getPayloadOrgToken(d *schema.ResourceData) *orgtoken.CreateUpdateTokenRequest {
	return &orgtoken.CreateUpdateTokenRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Disabled:    d.Get("disabled").(bool),
	}
}

func orgTokenCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadOrgToken(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Org Token Payload: %s", string(debugOutput))

	t, err := config.Client.CreateOrgToken(payload)
	if err != nil {
		return err
	}
	d.SetId(t.Name)
	return orgTokenAPIToTF(d, t)
}

func orgTokenAPIToTF(d *schema.ResourceData, t *orgtoken.Token) error {
	debugOutput, _ := json.Marshal(t)
	log.Printf("[DEBUG] SignalFx: Got Org Token to enState: %s", string(debugOutput))

	if err := d.Set("name", t.Name); err != nil {
		return err
	}
	if err := d.Set("description", t.Description); err != nil {
		return err
	}
	if err := d.Set("disabled", t.Disabled); err != nil {
		return err
	}

	return nil
}

func orgTokenRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	t, err := config.Client.GetOrgToken(d.Id())
	if err != nil {
		return err
	}

	return orgTokenAPIToTF(d, t)
}

func orgTokenUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadOrgToken(d)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Org Token Payload: %s", string(debugOutput))

	t, err := config.Client.UpdateOrgToken(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Org Token Response: %v", t)

	d.SetId(t.Name)
	return orgTokenAPIToTF(d, t)
}

func orgTokenDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteOrgToken(d.Id())
}

func orgTokenExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetOrgToken(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
