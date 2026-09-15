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
