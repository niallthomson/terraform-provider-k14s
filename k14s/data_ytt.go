package k14s

import (
	"crypto/sha256"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	cmdcore "github.com/k14s/ytt/pkg/cmd/core"
	"github.com/k14s/ytt/pkg/cmd/template"
	filespkg "github.com/k14s/ytt/pkg/files"
	"github.com/k14s/ytt/pkg/workspace"
)

func datasourceYtt() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"yaml": {
				Type:        schema.TypeString,
				Description: "Configuration yaml",
				Optional:    true,
			},
			"files": {
				Type:        schema.TypeList,
				Description: "List of configuration yaml files",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"values": {
				Type:        schema.TypeMap,
				Description: "Data values",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"result": {
				Type:        schema.TypeString,
				Description: "Rendered yaml",
				Computed:    true,
				Sensitive:   true,
			},
		},
		Read: resourceYttRead,
	}
}

func resourceYttRead(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml").(string)

	var valuesList []string

	if l, ok := d.GetOk("values"); ok {
		for k, v := range l.(map[string]interface{}) {
			valuesList = append(valuesList, k+"="+v.(string))
		}
	}

	ui := cmdcore.NewPlainUI(false)

	var filePaths []string

	filesParam := d.Get("files").([]interface{})
	if len(filesParam) > 0 {

		for _, fileParam := range filesParam {
			filePaths = append(filePaths, fileParam.(string))
		}
	}

	files, err := filespkg.NewSortedFilesFromPaths(filePaths, filespkg.SymlinkAllowOpts{
		AllowAll:        true,
		AllowedDstPaths: nil,
	})

	inlineFile, err := filespkg.NewFileFromSource(filespkg.NewCachedSource(filespkg.NewBytesSource("inline.yml", []byte(yaml))))
	if err != nil {
		return err
	}

	files = filespkg.NewSortedFiles(append(files, inlineFile))

	if err != nil {
		return err
	}

	rootLibrary := workspace.NewRootLibrary(files)
	rootLibrary.Print(ui.DebugWriter())

	libraryExecutionFactory := workspace.NewLibraryExecutionFactory(ui, workspace.TemplateLoaderOpts{
		IgnoreUnknownComments: true,
		StrictYAML:            false,
	})

	libraryCtx := workspace.LibraryExecutionContext{Current: rootLibrary, Root: rootLibrary}
	libraryLoader := libraryExecutionFactory.New(libraryCtx)

	dvFlags := template.DataValuesFlags{
		KVsFromStrings: valuesList,
	}

	valuesOverlays, err := dvFlags.AsOverlays(false)
	if err != nil {
		return err
	}

	values, err := libraryLoader.Values(valuesOverlays)
	if err != nil {
		return err
	}

	result, err := libraryLoader.Eval(values)
	if err != nil {
		return err
	}

	resultBytes, err := result.DocSet.AsBytes()
	if err != nil {
		return err
	}

	stdout := string(resultBytes)

	h := sha256.New()

	d.SetId(string(h.Sum(resultBytes)))
	d.Set("result", stdout)

	return nil
}
