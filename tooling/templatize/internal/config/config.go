package config

import (
	"bytes"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/Azure/ARO-HCP/tooling/templatize/internal/naming"
)

type Provider interface {
	GetVariables(cloud, deployEnv string) (Variables, error)
}

func NewConfigProvider(config, region, regionStamp, cxStamp string) *configProviderImpl {
	return &configProviderImpl{
		config:      config,
		region:      region,
		regionStamp: regionStamp,
		cxStamp:     cxStamp,
	}
}

// get the variables toke effect finally for cloud/deployEnv/region
func (cp *configProviderImpl) GetVariables(cloud, deployEnv string) (Variables, error) {
	variableOverrides, err := cp.loadConfig(cloud, deployEnv)
	variables := Variables{}

	if err == nil {
		for k, v := range variableOverrides.Defaults {
			variables[k] = v
		}
		if cloudOverride, ok := variableOverrides.Overrides[cloud]; ok {
			for k, v := range cloudOverride.Defaults {
				variables[k] = v
			}

			if deployEnvOverride, ok := cloudOverride.Overrides[deployEnv]; ok {
				for k, v := range deployEnvOverride.Defaults {
					variables[k] = v
				}

				if regionOverride, ok := deployEnvOverride.Overrides[cp.region]; ok {
					for k, v := range regionOverride {
						variables[k] = v
					}
				}
			}
		}
	}

	return variables, err
}

func (cp *configProviderImpl) loadConfig(cloud, deployEnv string) (*VariableOverrides, error) {
	vars := map[string]interface{}{
		"ctx": map[string]interface{}{
			"region":      cp.region,
			"cloud":       cloud,
			"deployEnv":   deployEnv,
			"regionStamp": cp.regionStamp,
			"cxStamp":     cp.cxStamp,
		},
	}

	functions := template.FuncMap{
		"azureEventGridName": naming.AzureEventGridName,
		"azurePostgresName":  naming.AzurePostgresName,
		"azureKeyVaultName":  naming.AzureKeyVaultName,
	}

	// parse, execute and unmarshal the config file as a template to generate the final config file
	tmpl := template.New("configTemplate").Funcs(functions)
	content, err := os.ReadFile(cp.config)
	if err != nil {
		return nil, err
	}

	tmpl, err = tmpl.Parse(string(content))
	if err != nil {
		return nil, err
	}

	var tmplBytes bytes.Buffer
	if err := tmpl.Execute(&tmplBytes, vars); err != nil {
		return nil, err
	}

	currentVariableOverrides := &VariableOverrides{}
	if err := yaml.Unmarshal(tmplBytes.Bytes(), currentVariableOverrides); err == nil {
		cp.baseVariableOverrides = currentVariableOverrides
	}

	return cp.baseVariableOverrides, err
}
