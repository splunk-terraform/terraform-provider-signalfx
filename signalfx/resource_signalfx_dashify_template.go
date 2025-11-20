// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	TemplateAPIPath = "/v2/template"
)

func dashifyTemplateResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the dashify template",
			},
			"template_contents": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "JSON contents of the dashify template",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					var js interface{}
					if err := json.Unmarshal([]byte(v), &js); err != nil {
						errs = append(errs, fmt.Errorf("%q must be valid JSON: %s", key, err))
					}
					return
				},
			},
		},
		Create: dashifyTemplateCreate,
		Read:   dashifyTemplateRead,
		Update: dashifyTemplateUpdate,
		Delete: dashifyTemplateDelete,
		Exists: dashifyTemplateExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func dashifyTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	// Parse the template contents
	templateContents := d.Get("template_contents").(string)

	// Make API request
	url := fmt.Sprintf("%s%s", config.APIURL, TemplateAPIPath)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(templateContents)))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SF-TOKEN", config.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making API request: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %s", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error creating template: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response to get the template ID
	// The API returns: {"data": {"id": "...", ...}, "errors": [], "includes": []}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("error parsing response: %s", err)
	}

	// Extract the data object
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("data field not found in response: %s", string(body))
	}

	// Extract the ID from data.id
	templateID, ok := data["id"].(string)
	if !ok {
		return fmt.Errorf("template ID not found in response data: %s", string(body))
	}

	d.SetId(templateID)
	log.Printf("[DEBUG] SignalFx: Created Dashify Template: %s", templateID)

	return dashifyTemplateRead(d, meta)
}

func dashifyTemplateRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	templateID := d.Id()
	url := fmt.Sprintf("%s%s/%s", config.APIURL, TemplateAPIPath, templateID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %s", err)
	}

	req.Header.Set("X-SF-TOKEN", config.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making API request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] SignalFx: Dashify Template not found: %s", templateID)
		d.SetId("")
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error reading template: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response - GET may return data in same format as POST
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("error parsing response: %s", err)
	}

	// Check if response has a "data" wrapper
	var templateData map[string]interface{}
	if data, ok := result["data"].(map[string]interface{}); ok {
		// Response is wrapped in {"data": {...}}
		templateData = data
	} else {
		// Response is direct template data
		templateData = result
	}

	// Set the template contents - marshal back to JSON
	templateJSON, err := json.Marshal(templateData)
	if err != nil {
		return fmt.Errorf("error marshaling template: %s", err)
	}

	if err := d.Set("template_contents", string(templateJSON)); err != nil {
		return err
	}

	// Set name if present
	if title, ok := templateData["title"].(string); ok {
		if err := d.Set("name", title); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] SignalFx: Read Dashify Template: %s", templateID)

	return nil
}

func dashifyTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	templateID := d.Id()
	templateContents := d.Get("template_contents").(string)

	// Make API request to update
	url := fmt.Sprintf("%s%s/%s", config.APIURL, TemplateAPIPath, templateID)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer([]byte(templateContents)))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SF-TOKEN", config.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making API request: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error updating template: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Printf("[DEBUG] SignalFx: Updated Dashify Template: %s", templateID)

	return dashifyTemplateRead(d, meta)
}

func dashifyTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	templateID := d.Id()
	url := fmt.Sprintf("%s%s/%s", config.APIURL, TemplateAPIPath, templateID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %s", err)
	}

	req.Header.Set("X-SF-TOKEN", config.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making API request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting template: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Printf("[DEBUG] SignalFx: Deleted Dashify Template: %s", templateID)

	return nil
}

func dashifyTemplateExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)

	templateID := d.Id()
	url := fmt.Sprintf("%s%s/%s", config.APIURL, TemplateAPIPath, templateID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating HTTP request: %s", err)
	}

	req.Header.Set("X-SF-TOKEN", config.AuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error making API request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("error checking template existence: status %d, body: %s", resp.StatusCode, string(body))
	}

	return true, nil
}
