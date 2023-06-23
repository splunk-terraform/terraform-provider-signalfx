package signalfx

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"reflect"
	"strings"
)

func handleIntegrationExists(err error) (bool, error) {
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func handleIntegrationRead(err error, d *schema.ResourceData) bool {
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return false
	}
	return true
}

func handleIntegrationChange(err error, d *schema.ResourceData, in interface{}) bool {
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return false
	}
	d.SetId(reflect.ValueOf(in).Elem().FieldByName("Id").String())
	return true
}

func logIntegrationData(format string, serviceName string, out interface{}) {
	debugOutput, _ := json.Marshal(out)
	log.Printf(format, serviceName, string(debugOutput))
}

func logIntegrationResponse(in interface{}, serviceName string) {
	logIntegrationData("[DEBUG] SignalFx: Got %s Integration to enState: %s", serviceName, in)
}

func logIntegrationCreateRequest(out interface{}, serviceName string) {
	logIntegrationData("[DEBUG] SignalFx: Create %s Integration Payload: %s", serviceName, out)
}

func logIntegrationUpdateRequest(out interface{}, serviceName string) {
	logIntegrationData("[DEBUG] SignalFx: Update %s Integration Payload: %s", serviceName, out)
}
