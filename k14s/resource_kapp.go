package k14s

import (
	"log"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/k14s/kapp/pkg/kapp/cmd/app"
	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
	ctlres "github.com/k14s/kapp/pkg/kapp/resources"
	"github.com/niallthomson/terraform-provider-k14s/k14s/kapp"
	util "github.com/niallthomson/terraform-provider-k14s/k14s/util"
)

func resourceApp() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the app",
				Required:    true,
				ForceNew:    true,
			},
			"namespace": {
				Type:        schema.TypeString,
				Description: "The default namespace to operate in",
				Required:    true,
				ForceNew:    true,
			},
			"yaml": {
				Type:        schema.TypeString,
				Description: "The config yaml to deploy",
				Optional:    true,
			},
			"files": {
				Type:        schema.TypeList,
				Description: "The yaml files to deploy",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		Create: resourceAppCreate,
		Read:   resourceAppRead,
		Update: resourceAppUpdate,
		Delete: resourceAppDelete,
		Exists: resourceAppExists,
	}
}

func resourceAppCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceAppDeploy(d, meta)
	if err != nil {
		return err
	}

	id := uuid.New().String()

	d.SetId(id)

	return nil
}

func resourceAppUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAppDeploy(d, meta)
}

func resourceAppDeploy(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Config)

	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	yaml := d.Get("yaml").(string)

	var files []string

	filesParam := d.Get("files").([]interface{})
	if len(filesParam) > 0 {
		for _, fileParam := range filesParam {
			files = append(files, fileParam.(string))
		}
	}

	err := kapp.NewDeployRequest(c.ConfigFactory, name, namespace, yaml, files).Execute()
	if err != nil {
		return err
	}

	return nil
}

func resourceAppDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Config)

	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

	err := kapp.NewDeleteRequest(c.ConfigFactory, name, namespace).Execute()
	if err != nil {
		return err
	}

	return nil
}

func resourceAppRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAppExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	log.Printf("Calling exists")

	c := meta.(*Config)

	name := d.Get("name").(string)

	failingAPIServicesPolicy := &app.FailingAPIServicesPolicy{}

	logger := util.NewStdOutLogger()

	ui := ui.NewConfUI(ui.NewNoopLogger())

	defer ui.Flush()

	depsFactory := util.NewDepsFactoryImpl(c.ConfigFactory)

	app, supportObjs, err := app.AppFactory(depsFactory, app.AppFlags{
		Name: name,
		NamespaceFlags: cmdcore.NamespaceFlags{
			Name: "default",
		},
	}, app.ResourceTypesFlags{}, logger)
	if err != nil {
		return false, err
	}

	usedGVs, err := app.UsedGVs()
	if err != nil {
		return false, err
	}

	failingAPIServicesPolicy.MarkRequiredGVs(usedGVs)

	labelSelector, err := app.LabelSelector()
	if err != nil {
		return false, err
	}

	resources, err := supportObjs.IdentifiedResources.List(labelSelector)
	if err != nil {
		return false, err
	}

	resourceFilter := ctlres.ResourceFilter{}

	resources = resourceFilter.Apply(resources)

	//cmdtools.InspectView{Source: source, Resources: resources, Sort: true}.Print(o.ui)

	log.Printf("Assuming app exists now")

	return true, nil
}
