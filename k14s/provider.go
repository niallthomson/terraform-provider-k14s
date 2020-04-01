package k14s

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
	util "github.com/niallthomson/terraform-provider-k14s/k14s/util"
)

const (
	schemaKappKey               = "kapp"
	schemaKappKubeconfigKey     = "kubeconfig"
	schemaKappKubeconfigYAMLKey = "kubeconfig_yaml"

	schemaKappKubeconfigFromEnvKey    = "from_env"
	schemaKappKubeconfigContextKey    = "context"
	schemaKappKubeconfigServerKey     = "server"
	schemaKappKubeconfigUsernameKey   = "username"
	schemaKappKubeconfigPasswordKey   = "password"
	schemaKappKubeconfigCACertKey     = "ca_cert"
	schemaKappKubeconfigClientCertKey = "client_cert"
	schemaKappKubeconfigClientKeyKey  = "client_key"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"kapp": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Kapp options",
				Optional:    true,
				MinItems:    0,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						schemaKappKubeconfigKey: {
							Type:        schema.TypeList,
							Description: "kubeconfig used by kapp",
							Optional:    true,
							MinItems:    0,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									schemaKappKubeconfigFromEnvKey: {
										Type:        schema.TypeBool,
										Description: "Pull configuration from environment (typically found in ~/.kube/config or via $KUBECONFIG)",
										Optional:    true,
									},
									schemaKappKubeconfigContextKey: {
										Type:        schema.TypeString,
										Description: "Use particular context",
										Optional:    true,
									},
									schemaKappKubeconfigServerKey: {
										Type:        schema.TypeString,
										Description: "Address of API server",
										Optional:    true,
									},
									schemaKappKubeconfigUsernameKey: {
										Type:        schema.TypeString,
										Description: "Username",
										Optional:    true,
									},
									schemaKappKubeconfigPasswordKey: {
										Type:        schema.TypeString,
										Description: "Password",
										Optional:    true,
									},
									schemaKappKubeconfigCACertKey: {
										Type:        schema.TypeString,
										Description: "CA certificate in PEM format",
										Optional:    true,
									},
									schemaKappKubeconfigClientCertKey: {
										Type:        schema.TypeString,
										Description: "Client certificate in PEM format",
										Optional:    true,
									},
									schemaKappKubeconfigClientKeyKey: {
										Type:        schema.TypeString,
										Description: "Client key in PEM format",
										Optional:    true,
									},
								},
							},
						},
						"kubeconfig_yaml": {
							Type:        schema.TypeString,
							Description: "kubeconfig as YAML",
							Optional:    true,
						},
					},
				},
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"k14sx_kapp": resourceApp(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"k14sx_ytt": datasourceYtt(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	kubeconfigYml := ""

	if val, ok := d.GetOkExists("kapp.0.kubeconfig_yaml"); ok {
		log.Println("Loading kubeconfig from yaml")

		kubeconfigYml = val.(string)
	} else {
		log.Println("Defaulting to context kubeconfig")
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
