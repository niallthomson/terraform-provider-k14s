package k14s

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/mitchellh/go-homedir"
	util "github.com/niallthomson/terraform-provider-k14s/k14s/util"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
						"kubernetes": {
							Type:        schema.TypeList,
							Description: "kubeconfig used by kapp",
							Optional:    true,
							MinItems:    0,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_HOST", ""),
										Description: "The hostname (in form of URI) of Kubernetes master.",
									},
									"username": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_USER", ""),
										Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
									},
									"password": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_PASSWORD", ""),
										Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
									},
									"insecure": {
										Type:        schema.TypeBool,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_INSECURE", false),
										Description: "Whether server should be accessed without verifying the TLS certificate.",
									},
									"client_certificate": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_CLIENT_CERT_DATA", ""),
										Description: "PEM-encoded client certificate for TLS authentication.",
									},
									"client_key": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_CLIENT_KEY_DATA", ""),
										Description: "PEM-encoded client certificate key for TLS authentication.",
									},
									"cluster_ca_certificate": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_CLUSTER_CA_CERT_DATA", ""),
										Description: "PEM-encoded root certificates bundle for TLS authentication.",
									},
									"config_path": {
										Type:     schema.TypeString,
										Optional: true,
										DefaultFunc: schema.MultiEnvDefaultFunc(
											[]string{
												"KUBE_CONFIG",
												"KUBECONFIG",
											},
											"~/.kube/config"),
										Description: "Path to the kube config file, defaults to ~/.kube/config",
									},
									"config_context": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX", ""),
									},
									"config_context_auth_info": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX_AUTH_INFO", ""),
										Description: "",
									},
									"config_context_cluster": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX_CLUSTER", ""),
										Description: "",
									},
									"token": {
										Type:        schema.TypeString,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_TOKEN", ""),
										Description: "Token to authenticate an service account",
									},
									"load_config_file": {
										Type:        schema.TypeBool,
										Optional:    true,
										DefaultFunc: schema.EnvDefaultFunc("KUBE_LOAD_CONFIG_FILE", true),
										Description: "Load local kubeconfig.",
									},
									"exec": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"api_version": {
													Type:     schema.TypeString,
													Required: true,
												},
												"command": {
													Type:     schema.TypeString,
													Required: true,
												},
												"env": {
													Type:     schema.TypeMap,
													Optional: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
												"args": {
													Type:     schema.TypeList,
													Optional: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
											},
										},
										Description: "",
									},
								},
							},
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
	clientConfig, err := initializeConfiguration(d)
	if err != nil {
		return nil, err
	}

	depsFactory := util.NewDepsFactoryImpl(clientConfig)

	config := &Config{
		DepsFactory: depsFactory,
	}

	return config, nil
}

// Copied this from kubernetes provider implementation
func initializeConfiguration(d *schema.ResourceData) (*restclient.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	if k8sGet(d, "load_config_file").(bool) {
		log.Printf("[DEBUG] Trying to load configuration from file")
		if configPath, ok := k8sGetOk(d, "config_path"); ok && configPath.(string) != "" {
			path, err := homedir.Expand(configPath.(string))
			if err != nil {
				return nil, err
			}
			log.Printf("[DEBUG] Configuration file is: %s", path)
			loader.ExplicitPath = path

			ctxSuffix := "; default context"

			ctx, ctxOk := k8sGetOk(d, "config_context")
			authInfo, authInfoOk := k8sGetOk(d, "config_context_auth_info")
			cluster, clusterOk := k8sGetOk(d, "config_context_cluster")
			if ctxOk || authInfoOk || clusterOk {
				ctxSuffix = "; overriden context"
				if ctxOk {
					overrides.CurrentContext = ctx.(string)
					ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
					log.Printf("[DEBUG] Using custom current context: %q", overrides.CurrentContext)
				}

				overrides.Context = clientcmdapi.Context{}
				if authInfoOk {
					overrides.Context.AuthInfo = authInfo.(string)
					ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
				}
				if clusterOk {
					overrides.Context.Cluster = cluster.(string)
					ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
				}
				log.Printf("[DEBUG] Using overidden context: %#v", overrides.Context)
			}
		}
	}

	// Overriding with static configuration
	if v, ok := k8sGetOk(d, "insecure"); ok {
		overrides.ClusterInfo.InsecureSkipTLSVerify = v.(bool)
	}
	if v, ok := k8sGetOk(d, "cluster_ca_certificate"); ok {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(d, "client_certificate"); ok {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(d, "host"); ok {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := restclient.DefaultServerURL(v.(string), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if v, ok := k8sGetOk(d, "username"); ok {
		overrides.AuthInfo.Username = v.(string)
	}
	if v, ok := k8sGetOk(d, "password"); ok {
		overrides.AuthInfo.Password = v.(string)
	}
	if v, ok := k8sGetOk(d, "client_key"); ok {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(d, "token"); ok {
		overrides.AuthInfo.Token = v.(string)
	}

	if v, ok := k8sGetOk(d, "exec"); ok {
		exec := &clientcmdapi.ExecConfig{}
		if spec, ok := v.([]interface{})[0].(map[string]interface{}); ok {
			exec.APIVersion = spec["api_version"].(string)
			exec.Command = spec["command"].(string)
			exec.Args = expandStringSlice(spec["args"].([]interface{}))
			for kk, vv := range spec["env"].(map[string]interface{}) {
				exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: kk, Value: vv.(string)})
			}
		} else {
			return nil, fmt.Errorf("Failed to parse exec")
		}
		overrides.AuthInfo.Exec = exec
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()
	if err != nil {
		log.Printf("[WARN] Invalid provider configuration was supplied. Provider operations likely to fail.")
		return nil, nil
	}

	log.Printf("[INFO] Successfully initialized config")
	return cfg, nil
}

func expandStringSlice(s []interface{}) []string {
	result := make([]string, len(s), len(s))
	for k, v := range s {
		// Handle the Terraform parser bug which turns empty strings in lists to nil.
		if v == nil {
			result[k] = ""
		} else {
			result[k] = v.(string)
		}
	}
	return result
}

var k8sPrefix = "kapp.0.kubernetes.0."

func k8sGetOk(d *schema.ResourceData, key string) (interface{}, bool) {
	value, ok := d.GetOk(k8sPrefix + key)

	// For boolean attributes the zero value is Ok
	switch value.(type) {
	case bool:
		value, ok = d.GetOkExists(k8sPrefix + key)
	}

	// removed for now
	// fix: DefaultFunc is not being triggerred on TypeList
	/*schema := kubernetesResource().Schema[key]
	if !ok && schema.DefaultFunc != nil {
		value, _ = schema.DefaultFunc()

		switch v := value.(type) {
		case string:
			ok = len(v) != 0
		case bool:
			ok = v
		}
	}*/

	return value, ok
}

func k8sGet(d *schema.ResourceData, key string) interface{} {
	value, _ := k8sGetOk(d, key)
	return value
}
