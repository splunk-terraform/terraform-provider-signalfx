package signalfx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	organization "github.com/signalfx/signalfx-go/organization"
)

const (
	MemberAppPath = "/organization/member/"
)

func memberResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"full_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Full name of the member",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Email address of the member",
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the member",
			},
		},

		Create: memberCreate,
		Read:   memberRead,
		Update: memberUpdate,
		Delete: memberDelete,
		Exists: memberExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create a member
*/
func getPayloadMember(d *schema.ResourceData) (*organization.CreateUpdateMemberRequest, error) {

	imr := &organization.CreateUpdateMemberRequest{
		FullName: d.Get("full_name").(string),
		Email:    d.Get("email").(string),
	}

	return imr, nil
}

func memberCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadMember(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Member Payload: %s", string(debugOutput))

	member, err := config.Client.InviteMember(payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, MemberAppPath+member.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(member.Id)

	return memberAPIToTF(d, member)
}

func memberExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetMember(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "Bad status 404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func memberRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	member, err := config.Client.GetMember(d.Id())
	if err != nil {
		if strings.HasPrefix(err.Error(), "Bad status 404") {
			d.SetId("")
		}
		return err
	}
	return memberAPIToTF(d, member)
}

func memberAPIToTF(d *schema.ResourceData, member *organization.Member) error {
	debugOutput, _ := json.Marshal(member)
	log.Printf("[DEBUG] SignalFx: Got Member to enState: %s", string(debugOutput))

	if err := d.Set("full_name", member.FullName); err != nil {
		return err
	}
	if err := d.Set("email", member.Email); err != nil {
		return err
	}

	return nil
}

func memberUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadMember(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Member Payload: %s", string(debugOutput))

	member, err := config.Client.UpdateMember(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Member Response: %v", member)
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, MemberAppPath+member.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(member.Id)
	return memberAPIToTF(d, member)
}

func memberDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteMember(d.Id())
}
