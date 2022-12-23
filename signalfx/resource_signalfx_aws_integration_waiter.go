package signalfx

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	sfx "github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/integration"
	"strconv"
	"time"
)

type TargetStateWaiter struct {
	client        *sfx.Client
	integrationId string

	timeout time.Duration
	delay   time.Duration

	ignoreFailedCancel bool
}

type TargetState struct {
	Name   string
	Target interface{}
}

type stateTarget struct {
	pending []string
	target  []string
}

var integrationStateTargets = map[interface{}]stateTarget{
	true: {
		pending: []string{"false"},
		target:  []string{"true"},
	},
	false: {
		pending: []string{"true"},
		target:  []string{"", "false"},
	},
}

var syncStateTargets = map[interface{}]stateTarget{
	"enabled": {
		pending: []string{"DISABLED", "CANCELLING", "CANCELLATION_FAILED"},
		target:  []string{"ENABLED"},
	},
	"disabled": {
		pending: []string{"ENABLED", "CANCELLING"},
		target:  []string{"", "DISABLED"},
	},
}

var stateTargetMapping = map[string]map[interface{}]stateTarget{
	"enabled":                   integrationStateTargets,
	"enable_logs_sync":          syncStateTargets,
	"log_sync_state":            syncStateTargets,
	"use_metric_streams_sync":   syncStateTargets,
	"metric_streams_sync_state": syncStateTargets,
}

var stateFuncMapping = map[string]stateSupplierFunc{
	"enabled":                   integrationStateSupplier,
	"enable_logs_sync":          logsSyncStateSupplier,
	"log_sync_state":            logsSyncStateSupplier,
	"use_metric_streams_sync":   metricStreamsStateSupplier,
	"metric_streams_sync_state": metricStreamsStateSupplier,
}

func NewWaiter(d *schema.ResourceData, config *signalfxConfig, integrationId string) *TargetStateWaiter {
	ignoreFailedCancel, ok := d.GetOk("ignore_cancellation_failure")

	return &TargetStateWaiter{
		client:        config.Client,
		integrationId: integrationId,

		timeout: d.Timeout(schema.TimeoutUpdate) - time.Minute,
		delay:   10 * time.Second,

		ignoreFailedCancel: ok && ignoreFailedCancel.(bool),
	}
}

func (t *TargetStateWaiter) WaitForAll(props []TargetState) (int *integration.AwsCloudWatchIntegration, err error) {
	for _, waitFor := range props {
		if int, err = t.WaitFor(waitFor); err != nil {
			return
		}
	}
	return
}

func (t *TargetStateWaiter) WaitFor(p TargetState) (*integration.AwsCloudWatchIntegration, error) {
	stateTarget := stateTargetMapping[p.Name][p.Target]
	stateSupplier := stateFuncMapping[p.Name]

	if t.ignoreFailedCancel {
		stateSupplier = ignoreFailedCancellation(stateSupplier)
	}

	stateConf := &resource.StateChangeConf{
		Pending: stateTarget.pending,
		Target:  stateTarget.target,
		Refresh: func() (interface{}, string, error) {
			int, err := t.client.GetAWSCloudWatchIntegration(context.TODO(), t.integrationId)
			if err != nil {
				return 0, "", err
			}
			return int, stateSupplier(int), nil
		},
		Timeout:    t.timeout,
		Delay:      t.delay,
		MinTimeout: 5 * time.Second,
	}

	int, err := stateConf.WaitForState()
	if err != nil {
		return nil, fmt.Errorf("Error waiting for integration %s state for %s to become %s: %s",
			t.integrationId, p.Name, p.Target, err)
	}
	return int.(*integration.AwsCloudWatchIntegration), nil
}

type stateSupplierFunc = func(*integration.AwsCloudWatchIntegration) string

func ignoreFailedCancellation(supplierFunc stateSupplierFunc) stateSupplierFunc {
	return func(integration *integration.AwsCloudWatchIntegration) string {
		state := supplierFunc(integration)
		if state == "CANCELLATION_FAILED" {
			return "DISABLED"
		}
		return state
	}
}

func metricStreamsStateSupplier(int *integration.AwsCloudWatchIntegration) string {
	return int.MetricStreamsSyncState
}

func logsSyncStateSupplier(int *integration.AwsCloudWatchIntegration) string {
	return int.LogsSyncState
}

func integrationStateSupplier(int *integration.AwsCloudWatchIntegration) string {
	return strconv.FormatBool(int.Enabled)
}

func toTargetState(d *schema.ResourceData, name string) TargetState {
	return TargetState{name, d.Get(name)}
}
