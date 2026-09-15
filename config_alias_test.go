package pluginsdk_test

import (
	"encoding/json"
	"reflect"
	"testing"

	sdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
)

func TestConfigAliasesMatchCommonSDK(t *testing.T) {
	for name, pair := range map[string][2]reflect.Type{
		"config":  {reflect.TypeOf(sdk.ManifestConfig{}), reflect.TypeOf(plugin.Config{})},
		"section": {reflect.TypeOf(sdk.ConfigSection{}), reflect.TypeOf(plugin.ConfigSection{})},
		"field":   {reflect.TypeOf(sdk.ConfigField{}), reflect.TypeOf(plugin.ConfigField{})},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s is not a common SDK type alias", name)
		}
	}
	got := []string{sdk.ConfigKindBool, sdk.ConfigKindInt, sdk.ConfigKindString, sdk.ConfigKindStringList, sdk.ConfigKindEnum, sdk.ConfigKindSecret}
	want := []string{plugin.ConfigKindBool, plugin.ConfigKindInt, plugin.ConfigKindString, plugin.ConfigKindStringList, plugin.ConfigKindEnum, plugin.ConfigKindSecret}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("kind constants = %v, want %v", got, want)
	}
	// ManifestConfig avoids colliding with the existing server Config API.
	_ = sdk.Config{}
}

func TestConfigFieldsForwardingMatchesCommonSDK(t *testing.T) {
	type settings struct {
		Enabled bool     `json:"enabled" config:"default=true"`
		Limit   *int     `json:"limit" config:"min=0,max=10,restart"`
		Name    string   `json:"name" config:"label=Name"`
		Muted   []string `json:"muted"`
		Mode    string   `json:"mode" config:"enum=low|high,default=high"`
		Token   string   `json:"token" config:"secret,readonly"`
		Ignored float64  `json:"-"`
	}
	for name, input := range map[string]any{
		"value": settings{}, "pointer": &settings{}, "nil": nil,
		"not struct": 42,
		"invalid tag": struct {
			Value bool `config:"misspelled=true"`
		}{},
		"invalid kind": struct{ Value float64 }{},
		"invalid enum": struct {
			Value string `config:"enum=a|b,default=c"`
		}{},
	} {
		t.Run(name, func(t *testing.T) {
			got, gotErr := sdk.ConfigFieldsFromStruct(input)
			want, wantErr := plugin.ConfigFieldsFromStruct(input)
			if (gotErr == nil) != (wantErr == nil) || (gotErr != nil && gotErr.Error() != wantErr.Error()) {
				t.Fatalf("error = %v, want %v", gotErr, wantErr)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("fields = %#v, want %#v", got, want)
			}
			if gotErr != nil {
				return
			}
			manifest := sdk.Manifest{Config: &sdk.ManifestConfig{Sections: []sdk.ConfigSection{{ID: "preferences", Fields: got}}}}
			encoded, err := json.Marshal(manifest.Config)
			if err != nil {
				t.Fatal(err)
			}
			canonical, err := json.Marshal(plugin.Config{Sections: []plugin.ConfigSection{{ID: "preferences", Fields: want}}})
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != string(canonical) {
				t.Fatalf("config JSON differs: %s != %s", encoded, canonical)
			}
		})
	}
}

func TestProviderConfigAliasesMatchCommonSDK(t *testing.T) {
	if sdk.ConfigKindProvider != plugin.ConfigKindProvider {
		t.Fatal("provider kind differs from common SDK")
	}
	for _, tc := range []struct {
		tag   string
		valid bool
	}{
		{"provider=host.data.store,requires=transactions|blobs", true},
		{"requires=transactions|blobs,provider=host.data.store", true},
		{"requires=transactions", false},
		{"provider=host.*", false},
		{"provider=host.data.store,requires=a|a", false},
		{"provider=host.data.store,enum=x", false},
		{"enum=x,provider=host.data.store", false},
		{"secret,provider=host.data.store", false},
		{"provider=host.data.store,secret", false},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			typ := reflect.StructOf([]reflect.StructField{{Name: "Store", Type: reflect.TypeOf(""), Tag: reflect.StructTag(`json:"store" config:"` + tc.tag + `"`)}})
			input := reflect.New(typ).Interface()
			got, err := sdk.ConfigFieldsFromStruct(input)
			want, wantErr := plugin.ConfigFieldsFromStruct(input)
			if (err == nil) != tc.valid || (err == nil) != (wantErr == nil) {
				t.Fatalf("valid=%v errors=%v/%v", tc.valid, err, wantErr)
			}
			if err != nil {
				if err.Error() != wantErr.Error() {
					t.Fatalf("validation errors differ: %v/%v", err, wantErr)
				}
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("fields differ: %#v/%#v", got, want)
			}
			if len(got) != 1 || got[0].Kind != sdk.ConfigKindProvider || got[0].Point != "host.data.store" || !reflect.DeepEqual(got[0].Requires, []string{"transactions", "blobs"}) {
				t.Fatalf("provider metadata lost: %#v", got)
			}
		})
	}
}
