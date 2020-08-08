package config

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

func TestLoadConfig(t *testing.T) {
	tcs := map[string]struct {
		want *Config
	}{
		"ok": {&Config{Port: 8080, Organization: "KeisukeYamashita"}},
	}

	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			os.Setenv("PORT", "8080")
			os.Setenv("ORGANIZATION", "KeisukeYamashita")
			got, err := LoadConfig(context.Background())
			if err != nil {
				t.Errorf("failed: %v", err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("There are diffs(got(-), want(+)):%s", diff)
			}
		})
	}
}

func TestLoadConfig_withlookup(t *testing.T) {
	tcs := map[string]struct {
		shouldPass bool
		lookuper   envconfig.Lookuper
		want       *Config
	}{
		"ok": {
			true,
			envconfig.MapLookuper(map[string]string{
				"PORT":         "8080",
				"ORGANIZATION": "KeisukeYamashita",
			}),
			&Config{
				Port:         8080,
				Organization: "KeisukeYamashita",
			},
		},
		"missing required env": {
			false,
			envconfig.MapLookuper(map[string]string{}),
			&Config{
				Port: 8080,
			},
		},
	}

	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			var got Config
			err := envconfig.ProcessWith(context.Background(), &got, tc.lookuper)
			if err != nil && tc.shouldPass {
				t.Errorf("failed: %v", err)
			}

			if diff := cmp.Diff(&got, tc.want); diff != "" {
				if tc.shouldPass {
					t.Errorf("(-got, +want)\n%s", diff)
				}
			}
		})
	}
}

func TestAddress(t *testing.T) {
	tcs := map[string]struct {
		port int
		want string
	}{
		"ok": {8080, ":8080"},
	}

	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Port: tc.port,
			}

			if got := cfg.Address(); got != tc.want {
				t.Errorf("failed got:%s want:%s", got, tc.want)
			}
		})
	}
}
