package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// This is an arbtirary limit and could be changed. I just don't think it
// makes a ton of sense to find more than this number.
var PAGE_LIMIT = int32(100)

func dataSourceDimensionValues() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceReadSignalFxDimensionValue,
		Schema: map[string]*schema.Schema{
			"query": {
				Type:     schema.TypeString,
				Required: true,
			},
			// Computed values
			"values": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceReadSignalFxDimensionValue(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	query := d.Get("query").(string)

	log.Printf("[DEBUG] SignalFx: Requesting dimension search: query=%s, limit=%d", query, PAGE_LIMIT)
	resp, err := config.Client.SearchDimension(context.TODO(), query, "", int(PAGE_LIMIT), 0)
	if err != nil {
		return err
	}
	debugOutput, _ := json.Marshal(resp)
	log.Printf("[DEBUG] SignalFx: Dimension Search Response Payload: %s", string(debugOutput))

	if resp.Count == 0 {
		return nil
	}
	if resp.Count >= PAGE_LIMIT {
		return fmt.Errorf("This data source only allows <= %d dimensions", PAGE_LIMIT)
	}

	values := make([]string, resp.Count)
	valueIndex := 0
	pagesNeeded := int(math.Ceil(float64(resp.Count) / float64(PAGE_LIMIT)))
	log.Printf("[DEBUG] SignalFx: Pages needed = %d, %f, %f", pagesNeeded, float64(PAGE_LIMIT), float64(resp.Count))
	for i := 0; i < pagesNeeded; i++ {

		if i > 0 {
			// If this isn't the first in the loop, fetch the next batch
			offset := i + int(PAGE_LIMIT) + 1
			log.Printf("[DEBUG] SignalFx: Requesting dimension search: query=%s, limit=%d, offset=%d", query, PAGE_LIMIT, offset)
			resp, err = config.Client.SearchDimension(context.TODO(), query, "", int(PAGE_LIMIT), offset)
			if err != nil {
				return err
			}
		}
		for _, v := range resp.Results {
			values[valueIndex] = v.Value
			valueIndex++
		}
	}

	log.Printf("[DEBUG] SignalFx: Got dimensions: %#v", values)
	if err := d.Set("values", values); err != nil {
		return err
	}
	d.SetId(query)

	return nil
}
