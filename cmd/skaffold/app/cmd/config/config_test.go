/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	yaml "gopkg.in/yaml.v2"
)

var baseConfig = &Config{
	Global: &ContextConfig{
		DefaultRepo: "test-repository",
	},
	ContextConfigs: []*ContextConfig{
		{
			Kubecontext: "test-context",
			DefaultRepo: "context-local-repository",
		},
	},
}

var emptyConfig = &Config{}

func TestReadConfig(t *testing.T) {
	c, _ := yaml.Marshal(*baseConfig)
	cfg, teardown := testutil.TempFile(t, "config", c)
	defer teardown()

	var tests = []struct {
		filename    string
		expectedCfg *Config
		shouldErr   bool
	}{
		{
			filename:  "",
			shouldErr: true,
		},
		{
			filename:    cfg,
			expectedCfg: baseConfig,
			shouldErr:   false,
		},
	}
	for _, test := range tests {
		cfg, err := ReadConfigForFile(test.filename)

		testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedCfg, cfg)
	}
}

func TestSetAndUnsetConfig(t *testing.T) {
	dummyContext := "dummy_context"
	reset := testutil.Override(t, &kubecontext, dummyContext)
	defer reset()

	var tests = []struct {
		expectedSetCfg   *Config
		expectedUnsetCfg *Config
		description      string
		key              string
		value            string
		kubecontext      string
		global           bool
		shouldErrSet     bool
	}{
		{
			description: "set default repo",
			key:         "default-repo",
			value:       "value",
			kubecontext: "this_is_a_context",
			expectedSetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: "this_is_a_context",
						DefaultRepo: "value",
					},
				},
			},
			expectedUnsetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: "this_is_a_context",
					},
				},
			},
		},
		{
			description: "set local cluster",
			key:         "local-cluster",
			value:       "false",
			kubecontext: "this_is_a_context",
			expectedSetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext:  "this_is_a_context",
						LocalCluster: util.BoolPtr(false),
					},
				},
			},
			expectedUnsetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: "this_is_a_context",
					},
				},
			},
		},
		{
			description:  "set invalid local cluster",
			key:          "local-cluster",
			shouldErrSet: true,
			value:        "not-a-bool",
			expectedSetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: dummyContext,
					},
				},
			},
		},
		{
			description:  "set fake value",
			key:          "not_a_real_value",
			shouldErrSet: true,
			expectedSetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: dummyContext,
					},
				},
			},
		},
		{
			description: "set global default repo",
			key:         "default-repo",
			value:       "global",
			global:      true,
			expectedSetCfg: &Config{
				Global: &ContextConfig{
					DefaultRepo: "global",
				},
				ContextConfigs: []*ContextConfig{},
			},
			expectedUnsetCfg: &Config{
				Global:         &ContextConfig{},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "set global local cluster",
			key:         "local-cluster",
			value:       "true",
			global:      true,
			expectedSetCfg: &Config{
				Global: &ContextConfig{
					LocalCluster: util.BoolPtr(true),
				},
				ContextConfigs: []*ContextConfig{},
			},
			expectedUnsetCfg: &Config{
				Global:         &ContextConfig{},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "set insecure registries",
			key:         "insecure-registries",
			value:       "my.insecure.registry",
			kubecontext: "this_is_a_context",
			expectedSetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext:        "this_is_a_context",
						InsecureRegistries: []string{"my.insecure.registry"},
					},
				},
			},
			expectedUnsetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: "this_is_a_context",
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// create new config file
			c, _ := yaml.Marshal(*emptyConfig)

			cfg := t.TempFile("config", c)

			t.Override(&configFile, cfg)
			t.Override(&global, test.global)
			if test.kubecontext != "" {
				t.Override(&kubecontext, test.kubecontext)
			}

			// set specified value
			err := setConfigValue(test.key, test.value)
			actualConfig, cfgErr := readConfig()
			if cfgErr != nil {
				t.Error(cfgErr)
			}
			t.CheckErrorAndDeepEqual(test.shouldErrSet, err, test.expectedSetCfg, actualConfig)

			if test.shouldErrSet {
				// if we expect an error when setting, don't try and unset
				return
			}

			// unset the value
			err = unsetConfigValue(test.key)
			newConfig, cfgErr := readConfig()
			if cfgErr != nil {
				t.Error(cfgErr)
			}
			t.CheckErrorAndDeepEqual(false, err, test.expectedUnsetCfg, newConfig)
		})
	}
}
