package hostindex

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"

	routeapi "github.com/openshift/origin/pkg/route/apis/route"
)

func TestOldestFirst(t *testing.T) {
	test1 := newRoute("test", "1", 1, 1, routeapi.RouteSpec{Host: "test.com"})
	test2 := newRoute("test", "2", 11, 2, routeapi.RouteSpec{Host: "test.com"})
	test3a := newRoute("test", "3", 12, 3, routeapi.RouteSpec{Host: "test.com", Path: "/a"})
	other1 := newRoute("other", "1", 1, 4, routeapi.RouteSpec{Host: "test.com"})
	other2 := newRoute("other", "2", 11, 5, routeapi.RouteSpec{Host: "test.com"})

	type args struct {
		active   []*routeapi.Route
		inactive []*routeapi.Route
	}
	tests := []struct {
		name          string
		args          args
		wantUpdated   []*routeapi.Route
		wantDisplaced []*routeapi.Route
		activates     map[string]struct{}
		displaces     map[string]struct{}
	}{
		{
			name: "displacement",
			args: args{
				active:   []*routeapi.Route{test2},
				inactive: []*routeapi.Route{test1},
			},
			wantUpdated:   []*routeapi.Route{test1},
			activates:     map[string]struct{}{"001": {}},
			wantDisplaced: []*routeapi.Route{test2},
			displaces:     map[string]struct{}{"011": {}},
		},
		{
			name: "exclude identical route",
			args: args{
				active:   []*routeapi.Route{test1},
				inactive: []*routeapi.Route{test2},
			},
			wantUpdated:   []*routeapi.Route{test1},
			wantDisplaced: []*routeapi.Route{test2},
		},
		{
			name: "add newer path based route",
			args: args{
				active:   []*routeapi.Route{test1},
				inactive: []*routeapi.Route{test3a},
			},
			wantUpdated: []*routeapi.Route{test1, test3a},
			activates:   map[string]struct{}{"012": {}},
		},
		{
			name: "add older path based route",
			args: args{
				active:   []*routeapi.Route{test3a},
				inactive: []*routeapi.Route{test1},
			},
			wantUpdated: []*routeapi.Route{test1, test3a},
			activates:   map[string]struct{}{"001": {}},
		},
		{
			name: "add an older route in a different namespace",
			args: args{
				active:   []*routeapi.Route{test2, test3a},
				inactive: []*routeapi.Route{other1},
			},
			wantUpdated:   []*routeapi.Route{other1, test3a},
			activates:     map[string]struct{}{"001": {}},
			wantDisplaced: []*routeapi.Route{test2},
			displaces:     map[string]struct{}{"011": {}},
		},
		{
			// the input list must be sorted
			name: "add two out-of-order routes at once gives incorrect results",
			args: args{
				active:   []*routeapi.Route{other2},
				inactive: []*routeapi.Route{test3a, test1},
			},
			wantUpdated:   []*routeapi.Route{other2, test3a},
			activates:     map[string]struct{}{"012": {}},
			wantDisplaced: []*routeapi.Route{test1},
			displaces:     map[string]struct{}{},
		},
		{
			name: "add two routes at once",
			args: args{
				active:   []*routeapi.Route{other2},
				inactive: []*routeapi.Route{test1, test3a},
			},
			wantUpdated:   []*routeapi.Route{test1, test3a},
			activates:     map[string]struct{}{"001": {}, "012": {}},
			wantDisplaced: []*routeapi.Route{other2},
			displaces:     map[string]struct{}{"011": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.activates == nil {
				tt.activates = make(map[string]struct{})
			}
			if tt.displaces == nil {
				tt.displaces = make(map[string]struct{})
			}
			// ensure no one accidentally provides the same UID twice
			uids := make(map[string]struct{})
			for _, arg := range tt.args.active {
				if _, ok := uids[string(arg.UID)]; ok {
					t.Fatalf("test may not have duplicate UIDs")
				}
				uids[string(arg.UID)] = struct{}{}
			}

			changes := &routeChanges{}
			gotUpdated, gotDisplaced := OldestFirst(changes, tt.args.active, tt.args.inactive...)
			if !reflect.DeepEqual(gotUpdated, tt.wantUpdated) {
				t.Errorf("OldestFirst() updated: %s", diff.ObjectReflectDiff(tt.wantUpdated, gotUpdated))
			}
			if !reflect.DeepEqual(gotDisplaced, tt.wantDisplaced) {
				t.Errorf("OldestFirst() displaced: %s", diff.ObjectReflectDiff(tt.wantDisplaced, gotDisplaced))
			}

			activates := changesToMap(changes.GetActivated())
			if !reflect.DeepEqual(tt.activates, activates) {
				t.Errorf("Unexpected activated changes: %s", diff.ObjectReflectDiff(tt.activates, activates))
			}
			displaces := changesToMap(changes.GetDisplaced())
			if !reflect.DeepEqual(tt.displaces, displaces) {
				t.Errorf("Unexpected displaced changes: %s", diff.ObjectReflectDiff(tt.displaces, displaces))
			}

		})
	}
}

func TestSameNamespace(t *testing.T) {
	test1 := newRoute("test", "1", 1, 1, routeapi.RouteSpec{Host: "test.com"})
	test2 := newRoute("test", "2", 11, 2, routeapi.RouteSpec{Host: "test.com"})
	test3a := newRoute("test", "3", 12, 3, routeapi.RouteSpec{Host: "test.com", Path: "/a"})
	other1 := newRoute("other", "1", 1, 4, routeapi.RouteSpec{Host: "test.com"})
	other2 := newRoute("other", "2", 11, 5, routeapi.RouteSpec{Host: "test.com"})

	type args struct {
		active   []*routeapi.Route
		inactive []*routeapi.Route
	}
	tests := []struct {
		name          string
		args          args
		wantUpdated   []*routeapi.Route
		wantDisplaced []*routeapi.Route
		activates     map[string]struct{}
		displaces     map[string]struct{}
	}{
		{
			name: "empty",
			args: args{
				active:   []*routeapi.Route{},
				inactive: []*routeapi.Route{test1},
			},
			wantUpdated: []*routeapi.Route{test1},
			activates:   map[string]struct{}{"001": {}},
		},
		{
			name: "displacement",
			args: args{
				active:   []*routeapi.Route{test2},
				inactive: []*routeapi.Route{test1},
			},
			wantUpdated:   []*routeapi.Route{test1},
			activates:     map[string]struct{}{"001": {}},
			wantDisplaced: []*routeapi.Route{test2},
			displaces:     map[string]struct{}{"011": {}},
		},
		{
			name: "exclude identical route",
			args: args{
				active:   []*routeapi.Route{test1},
				inactive: []*routeapi.Route{test2},
			},
			wantUpdated:   []*routeapi.Route{test1},
			wantDisplaced: []*routeapi.Route{test2},
		},
		{
			name: "add newer path based route",
			args: args{
				active:   []*routeapi.Route{test1},
				inactive: []*routeapi.Route{test3a},
			},
			wantUpdated: []*routeapi.Route{test1, test3a},
			activates:   map[string]struct{}{"012": {}},
		},
		{
			name: "add older path based route",
			args: args{
				active:   []*routeapi.Route{test3a},
				inactive: []*routeapi.Route{test1},
			},
			wantUpdated: []*routeapi.Route{test1, test3a},
			activates:   map[string]struct{}{"001": {}},
		},
		{
			name: "add an older route in a different namespace",
			args: args{
				active:   []*routeapi.Route{test2, test3a},
				inactive: []*routeapi.Route{other1},
			},
			wantUpdated:   []*routeapi.Route{other1},
			activates:     map[string]struct{}{"001": {}},
			wantDisplaced: []*routeapi.Route{test2, test3a},
			displaces:     map[string]struct{}{"011": {}, "012": {}},
		},
		{
			// the input list must be sorted
			name: "add two out-of-order routes at once gives incorrect results",
			args: args{
				active:   []*routeapi.Route{other2},
				inactive: []*routeapi.Route{test3a, test1},
			},
			wantUpdated:   []*routeapi.Route{other2},
			activates:     map[string]struct{}{},
			wantDisplaced: []*routeapi.Route{test3a, test1},
			displaces:     map[string]struct{}{},
		},
		{
			name: "add two routes at once",
			args: args{
				active:   []*routeapi.Route{other2},
				inactive: []*routeapi.Route{test1, test3a},
			},
			wantUpdated:   []*routeapi.Route{test1, test3a},
			activates:     map[string]struct{}{"001": {}, "012": {}},
			wantDisplaced: []*routeapi.Route{other2},
			displaces:     map[string]struct{}{"011": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.activates == nil {
				tt.activates = make(map[string]struct{})
			}
			if tt.displaces == nil {
				tt.displaces = make(map[string]struct{})
			}
			// ensure no one accidentally provides the same UID twice
			uids := make(map[string]struct{})
			for _, arg := range tt.args.active {
				if _, ok := uids[string(arg.UID)]; ok {
					t.Fatalf("test may not have duplicate UIDs")
				}
				uids[string(arg.UID)] = struct{}{}
			}

			changes := &routeChanges{}
			gotUpdated, gotDisplaced := SameNamespace(changes, tt.args.active, tt.args.inactive...)
			if !reflect.DeepEqual(gotUpdated, tt.wantUpdated) {
				t.Errorf("SameNamespace() updated: %s", diff.ObjectReflectDiff(tt.wantUpdated, gotUpdated))
			}
			if !reflect.DeepEqual(gotDisplaced, tt.wantDisplaced) {
				t.Errorf("SameNamespace() displaced: %s", diff.ObjectReflectDiff(tt.wantDisplaced, gotDisplaced))
			}

			activates := changesToMap(changes.GetActivated())
			if !reflect.DeepEqual(tt.activates, activates) {
				t.Errorf("Unexpected activated changes: %s", diff.ObjectReflectDiff(tt.activates, activates))
			}
			displaces := changesToMap(changes.GetDisplaced())
			if !reflect.DeepEqual(tt.displaces, displaces) {
				t.Errorf("Unexpected displaced changes: %s", diff.ObjectReflectDiff(tt.displaces, displaces))
			}

		})
	}
}
