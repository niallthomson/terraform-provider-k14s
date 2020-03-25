package k14s

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
	util "github.com/niallthomson/terraform-provider-k14s/k14s/util"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"kubeconfig_yml": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"k14s_app": resourceApp(),
			/*"tmc_cluster":       resourceCluster(),
			"tmc_workspace":     resourceWorkspace(),
			"tmc_node_pool":     resourceNodePool(),
			"tmc_namespace":     resourceNamespace(),*/
		},

		DataSourcesMap: map[string]*schema.Resource{
			"k14s_ytt": datasourceYtt(),
			/*"tmc_kubeconfig":    datasourceKubeconfig(),
			"tmc_cluster_group": datasourceClusterGroup(),
			"tmc_workspace":     datasourceWorkspace(),
			"tmc_namespace":     datasourceNamespace(),*/
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	kubeconfigYml := ""

	if val, ok := d.GetOkExists("kubeconfig_yml"); ok {
		kubeconfigYml = val.(string)
	}

	configFactory := cmdcore.NewConfigFactoryImpl()
	configFactory.ConfigurePathResolver(util.Resolver(""))
	configFactory.ConfigureYAMLResolver(util.Resolver(kubeconfigYml))
	configFactory.ConfigureContextResolver(util.Resolver(""))

	config := &Config{
		ConfigFactory: configFactory,
	}

	return config, nil
}
