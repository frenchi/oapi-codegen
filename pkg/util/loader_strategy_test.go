package util

import (
	"reflect"
	"strings"
	"testing"
)

func TestGetLoaderStrategy_Selection(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		expect any
	}{
		{name: "default -> kin", env: "", expect: kinLoader{}},
		{name: "false -> kin (0)", env: "0", expect: kinLoader{}},
		{name: "false -> kin (FALSE)", env: "FALSE", expect: kinLoader{}},
		{name: "soft -> lib (1)", env: "1", expect: libopenapiLoader{}},
		{name: "soft -> lib (true)", env: "true", expect: libopenapiLoader{}},
		{name: "soft -> lib (YES)", env: "YES", expect: libopenapiLoader{}},
		{name: "strict -> lib", env: "strict", expect: libopenapiLoaderStrict{}},
		{name: "strict -> lib (STRICT)", env: "STRICT", expect: libopenapiLoaderStrict{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OAPI_CODEGEN_USE_LIBOPENAPI", tt.env)
			s := getLoaderStrategy()
			got := reflect.TypeOf(s).Name()
			exp := reflect.TypeOf(tt.expect).Name()
			if !strings.Contains(got, exp) {
				t.Fatalf("env=%q: expected %s, got %s", tt.env, exp, got)
			}
		})
	}
}
