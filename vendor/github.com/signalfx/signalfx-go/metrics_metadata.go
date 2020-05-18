package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/signalfx/signalfx-go/metrics_metadata"
)

// DimensionAPIURL is the base URL for interacting with dimensions.
const DimensionAPIURL = "/v2/dimension"

// MetricAPIURL is the base URL for interacting with dimensions.
const MetricAPIURL = "/v2/metric"

// MetricTimeSeriesAPIURL is the base URL for interacting with dimensions.
const MetricTimeSeriesAPIURL = "/v2/metrictimeseries"

// TagAPIURL is the base URL for interacting with dimensions.
const TagAPIURL = "/v2/tag"

// GetDimension gets a dimension.
func (c *Client) GetDimension(key string, value string) (*metrics_metadata.Dimension, error) {
	resp, err := c.doRequest("GET", DimensionAPIURL+"/"+key+"/"+value, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalDimension := &metrics_metadata.Dimension{}

	err = json.NewDecoder(resp.Body).Decode(finalDimension)

	return finalDimension, err
}

// UpdateDimension updates a dimension.
func (c *Client) UpdateDimension(key string, value string, dim *metrics_metadata.Dimension) (*metrics_metadata.Dimension, error) {
	payload, err := json.Marshal(dim)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", DimensionAPIURL+"/"+key+"/"+value, nil, bytes.NewReader(payload))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalDimension := &metrics_metadata.Dimension{}

	err = json.NewDecoder(resp.Body).Decode(finalDimension)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDimension, err
}

// SearchDimension searches for dimensions, given a query string in `query`.
func (c *Client) SearchDimension(query string, orderBy string, limit int, offset int) (*metrics_metadata.DimensionQueryResponseModel, error) {
	params := url.Values{}
	params.Add("query", query)
	if orderBy != "" {
		params.Add("orderBy", orderBy)
	}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", DimensionAPIURL, params, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalDimensions := &metrics_metadata.DimensionQueryResponseModel{}

	err = json.NewDecoder(resp.Body).Decode(finalDimensions)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDimensions, err
}

// SearchMetric searches for metrics, given a query string in `query`.
func (c *Client) SearchMetric(query string, orderBy string, limit int, offset int) (*metrics_metadata.RetrieveMetricMetadataResponseModel, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("orderBy", orderBy)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", MetricAPIURL, params, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalMetrics := &metrics_metadata.RetrieveMetricMetadataResponseModel{}

	err = json.NewDecoder(resp.Body).Decode(finalMetrics)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalMetrics, err
}

// GetMetric retrieves a single metric by name.
func (c *Client) GetMetric(name string) (*metrics_metadata.Metric, error) {
	resp, err := c.doRequest("GET", MetricAPIURL+"/"+name, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalMetric := &metrics_metadata.Metric{}

	err = json.NewDecoder(resp.Body).Decode(finalMetric)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalMetric, err
}

// GetMetricTimeSeries retrieves a metric time series by id.
func (c *Client) GetMetricTimeSeries(id string) (*metrics_metadata.MetricTimeSeries, error) {
	resp, err := c.doRequest("GET", MetricTimeSeriesAPIURL+"/"+id, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalMetricTimeSeries := &metrics_metadata.MetricTimeSeries{}

	err = json.NewDecoder(resp.Body).Decode(finalMetricTimeSeries)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalMetricTimeSeries, err
}

// SearchMetricTimeSeries searches for metric time series, given a query string in `query`.
func (c *Client) SearchMetricTimeSeries(query string, orderBy string, limit int, offset int) (*metrics_metadata.MetricTimeSeriesRetrieveResponseModel, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("orderBy", orderBy)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", MetricTimeSeriesAPIURL, params, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalMTS := &metrics_metadata.MetricTimeSeriesRetrieveResponseModel{}

	err = json.NewDecoder(resp.Body).Decode(finalMTS)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalMTS, err
}

// SearchTag searches for tags, given a query string in `query`.
func (c *Client) SearchTag(query string, orderBy string, limit int, offset int) (*metrics_metadata.TagRetrieveResponseModel, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("orderBy", orderBy)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", TagAPIURL, params, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalTags := &metrics_metadata.TagRetrieveResponseModel{}

	err = json.NewDecoder(resp.Body).Decode(finalTags)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalTags, err
}

// GetTag gets a tag by name
func (c *Client) GetTag(name string) (*metrics_metadata.Tag, error) {
	resp, err := c.doRequest("GET", TagAPIURL+"/"+name, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalTag := &metrics_metadata.Tag{}

	err = json.NewDecoder(resp.Body).Decode(finalTag)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalTag, err
}

// DeleteTag deletes a tag.
func (c *Client) DeleteTag(id string) error {
	resp, err := c.doRequest("DELETE", TagAPIURL+"/"+id, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		message, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return nil
}

// CreateUpdateTag creates or updates a dimension.
func (c *Client) CreateUpdateTag(name string, cutr *metrics_metadata.CreateUpdateTagRequest) (*metrics_metadata.Tag, error) {
	payload, err := json.Marshal(cutr)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", TagAPIURL+"/"+name, nil, bytes.NewReader(payload))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalTag := &metrics_metadata.Tag{}

	err = json.NewDecoder(resp.Body).Decode(finalTag)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalTag, err
}
