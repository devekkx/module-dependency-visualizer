package config_test

import (
	"strings"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/config"
)

func TestBuildInfo_String(t *testing.T) {
	cases := []struct {
		name string
		in   config.BuildInfo
		want []string
	}{
		{
			name: "dev defaults",
			in:   config.Default,
			want: []string{"version=dev", "commit=none", "built=unknown"},
		},
		{
			name: "release build",
			in:   config.BuildInfo{Version: "v1.2.3", Commit: "abc1234", BuildDate: "2026-05-06"},
			want: []string{"version=v1.2.3", "commit=abc1234", "built=2026-05-06"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.String()
			for _, fragment := range tc.want {
				if !strings.Contains(got, fragment) {
					t.Errorf("String() = %q; want it to contain %q", got, fragment)
				}
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if config.SchemaVersion == "" {
		t.Error("SchemaVersion must not be empty")
	}
	if config.DefaultTimeout <= 0 {
		t.Error("DefaultTimeout must be positive")
	}
	if config.DefaultDepth != -1 {
		t.Errorf("DefaultDepth = %d; want -1", config.DefaultDepth)
	}
}
