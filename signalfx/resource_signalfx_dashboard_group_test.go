package signalfx

import (
	"reflect"
	"testing"
)

func TestGetMirrorsToBeOmitted(t *testing.T) {
	type args struct {
		oldDashboardList interface{}
		newDashboardList interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]bool
	}{
		{
			"Same values result in empty omit list",
			args{
				oldDashboardList: []interface{}{
					map[string]interface{}{
						"dashboard_id": "A",
					},
				},
				newDashboardList: []interface{}{
					map[string]interface{}{
						"dashboard_id": "A",
					},
				},
			},
			map[string]bool{},
		},
		{
			"Omit values absent in new list",
			args{
				oldDashboardList: []interface{}{
					map[string]interface{}{
						"dashboard_id": "A",
					},
					map[string]interface{}{
						"dashboard_id": "B",
					},
				},
				newDashboardList: []interface{}{
					map[string]interface{}{
						"dashboard_id": "B",
					},
					map[string]interface{}{
						"dashboard_id": "C",
					},
				},
			},
			map[string]bool{
				"A": true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMirrorsToBeOmitted(tt.args.oldDashboardList, tt.args.newDashboardList); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMirrorsToBeOmitted() = %v, want %v", got, tt.want)
			}
		})
	}
}
