package kapp

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/cppforlife/go-cli-ui/ui"
	ctlapp "github.com/k14s/kapp/pkg/kapp/app"
	ctlcap "github.com/k14s/kapp/pkg/kapp/clusterapply"
	"github.com/k14s/kapp/pkg/kapp/cmd/app"
	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
	ctlconf "github.com/k14s/kapp/pkg/kapp/config"
	ctldiff "github.com/k14s/kapp/pkg/kapp/diff"
	ctldgraph "github.com/k14s/kapp/pkg/kapp/diffgraph"
	ctlres "github.com/k14s/kapp/pkg/kapp/resources"
	util "github.com/niallthomson/terraform-provider-k14s/k14s/util"
)

type DeployRequest struct {
	depsFactory cmdcore.DepsFactory
	name        string
	namespace   string
	yaml        string
	files       []string
}

func NewDeployRequest(depsFactory cmdcore.DepsFactory, name string, namespace string, yaml string, files []string) *DeployRequest {
	return &DeployRequest{
		depsFactory: depsFactory,
		name:        name,
		namespace:   namespace,
		yaml:        yaml,
		files:       files,
	}
}

func (r *DeployRequest) Execute() error {
	failingAPIServicesPolicy := &app.FailingAPIServicesPolicy{}

	logger := util.NewStdOutLogger()

	ui := &util.LoggingUI{}
	defer ui.Flush()

	app, supportObjs, err := app.AppFactory(r.depsFactory, app.AppFlags{
		Name: r.name,
		NamespaceFlags: cmdcore.NamespaceFlags{
			Name: r.namespace,
		},
	}, app.ResourceTypesFlags{}, logger)
	if err != nil {
		return err
	}

	appLabels := make(map[string]string)

	err = app.CreateOrUpdate(appLabels)
	if err != nil {
		return err
	}

	usedGVs, err := app.UsedGVs()
	if err != nil {
		return err
	}

	failingAPIServicesPolicy.MarkRequiredGVs(usedGVs)

	prepOpts := ctlapp.PrepareResourcesOpts{}
	prepOpts.DefaultNamespace = r.namespace
	prepOpts.BeforeModificationFunc = func(rs []ctlres.Resource) []ctlres.Resource {
		failingAPIServicesPolicy.MarkRequiredResources(rs)
		return rs
	}

	prep := ctlapp.NewPreparation(supportObjs.ResourceTypes, prepOpts)

	labelSelector, err := app.LabelSelector()
	if err != nil {
		return err
	}

	labeledResources := ctlres.NewLabeledResources(labelSelector, supportObjs.IdentifiedResources, logger)

	resourceFilter := ctlres.ResourceFilter{}

	newResources, conf, nsNames, err := r.newResources(prep, labeledResources, resourceFilter)
	if err != nil {
		return err
	}

	existingResources, err := r.existingResources(newResources, labeledResources, resourceFilter, supportObjs.Apps)
	if err != nil {
		return err
	}

	clusterChangeSet, clusterChangesGraph, hasNoChanges, changeSummary, err :=
		r.calculateAndPresentChanges(existingResources, newResources, conf, supportObjs, ui)
	if err != nil {
		return err
	}

	// Validate new resources _after_ presenting changes to make it easier to see big picture
	err = prep.ValidateResources(newResources)
	if err != nil {
		return err
	}

	if hasNoChanges {
		return nil
	}

	err = app.UpdateUsedGVs(failingAPIServicesPolicy.GVs(newResources, existingResources))
	if err != nil {
		return err
	}

	defer func() {
		// hacked in 200 below
		_, numDeleted, _ := app.GCChanges(200, nil)
		if numDeleted > 0 {
			log.Printf("Deleted %d older app changes", numDeleted)
		}
	}()

	touch := ctlapp.Touch{
		App:              app,
		Description:      "update: " + changeSummary,
		Namespaces:       nsNames,
		IgnoreSuccessErr: true,
	}

	err = touch.Do(func() error {
		err := clusterChangeSet.Apply(clusterChangesGraph)
		if err != nil {
			return err
		}
		return app.UpdateUsedGVs(failingAPIServicesPolicy.GVs(newResources, nil))
	})
	if err != nil {
		return err
	}

	/*if o.ApplyFlags.ExitStatus {
		return DeployApplyExitStatus{hasNoChanges}
	}*/
	return nil
}

func (r *DeployRequest) existingResources(newResources []ctlres.Resource,
	labeledResources *ctlres.LabeledResources, resourceFilter ctlres.ResourceFilter,
	apps ctlapp.Apps) ([]ctlres.Resource, error) {

	labelErrorResolutionFunc := func(key string, val string) string {
		items, _ := apps.List(nil)
		for _, item := range items {
			meta, _ := item.Meta()
			if meta.LabelKey == key && meta.LabelValue == val {
				return fmt.Sprintf("different %s (label '%s=%s')", item.Description(), key, val)
			}
		}
		return ""
	}

	matchingOpts := ctlres.AllAndMatchingOpts{
		SkipResourceOwnershipCheck: true,
		// Prevent accidently overriding kapp state records
		BlacklistedResourcesByLabelKeys: []string{ctlapp.KappIsAppLabelKey},
		LabelErrorResolutionFunc:        labelErrorResolutionFunc,
	}

	existingResources, err := labeledResources.AllAndMatching(newResources, matchingOpts)
	if err != nil {
		return nil, err
	}

	// patch?
	if false {
		existingResources, err = ctlres.NewUniqueResources(existingResources).Match(newResources)
		if err != nil {
			return nil, err
		}
	} else {
		if len(newResources) == 0 {
			return nil, fmt.Errorf("Trying to apply empty set of resources will result in deletion of resources on cluster. " +
				"Refusing to continue unless --dangerous-allow-empty-list-of-resources is specified.")
		}
	}

	return resourceFilter.Apply(existingResources), nil
}

func (r *DeployRequest) calculateAndPresentChanges(existingResources,
	newResources []ctlres.Resource, conf ctlconf.Conf, supportObjs app.AppFactorySupportObjs, ui ui.UI) (
	ctlcap.ClusterChangeSet, *ctldgraph.ChangeGraph, bool, string, error) {

	var clusterChangeSet ctlcap.ClusterChangeSet

	changeSetOpts := ctldiff.ChangeSetOpts{
		AgainstLastApplied: true,
	}

	// whats the actual defaults for all these?
	clusterChangeOpts := ctlcap.ClusterChangeOpts{
		ApplyIgnored: false,
		Wait:         true,
		WaitIgnored:  false,

		AddOrUpdateChangeOpts: ctlcap.AddOrUpdateChangeOpts{
			DefaultUpdateStrategy: "",
		},
	}

	clusterChangeSetOpts := ctlcap.ClusterChangeSetOpts{
		ApplyingChangesOpts: ctlcap.ApplyingChangesOpts{
			Concurrency: 5,
		},
		WaitingChangesOpts: ctlcap.WaitingChangesOpts{
			CheckInterval: 5 * time.Second,
			Timeout:       15 * time.Minute,
		},
	}

	changeSetViewOpts := ctlcap.ChangeSetViewOpts{
		Changes: true,
		Summary: true,
		TextDiffViewOpts: ctldiff.TextDiffViewOpts{
			Context: 1,
			Mask:    true,
		},
	}

	{ // Figure out changes for X existing resources -> X new resources
		changeFactory := ctldiff.NewChangeFactory(conf.RebaseMods(), conf.DiffAgainstLastAppliedFieldExclusionMods())
		changeSetFactory := ctldiff.NewChangeSetFactory(changeSetOpts, changeFactory)

		changes, err := ctldiff.NewChangeSetWithTemplates(
			existingResources, newResources, conf.TemplateRules(),
			changeSetOpts, changeFactory).Calculate()
		if err != nil {
			return clusterChangeSet, nil, false, "", err
		}

		msgsUI := cmdcore.NewDedupingMessagesUI(cmdcore.NewPlainMessagesUI(ui))

		convergedResFactory := ctlcap.NewConvergedResourceFactory(ctlcap.ConvergedResourceFactoryOpts{
			IgnoreFailingAPIServices: false,
		})

		clusterChangeFactory := ctlcap.NewClusterChangeFactory(
			clusterChangeOpts, supportObjs.IdentifiedResources,
			changeFactory, changeSetFactory, convergedResFactory, msgsUI)

		clusterChangeSet = ctlcap.NewClusterChangeSet(
			changes, clusterChangeSetOpts, clusterChangeFactory, msgsUI)
	}

	clusterChanges, clusterChangesGraph, err := clusterChangeSet.Calculate()
	if err != nil {
		return clusterChangeSet, nil, false, "", err
	}

	var changesSummary string

	{ // Present cluster changes in UI
		changeViews := ctlcap.ClusterChangesAsChangeViews(clusterChanges)
		changeSetView := ctlcap.NewChangeSetView(changeViews, conf.DiffMaskRules(), changeSetViewOpts)
		changeSetView.Print(ui)
		changesSummary = changeSetView.Summary()
	}

	return clusterChangeSet, clusterChangesGraph, (len(clusterChanges) == 0), changesSummary, err
}

func (r *DeployRequest) nsNames(resources []ctlres.Resource) []string {
	uniqNames := map[string]struct{}{}
	names := []string{}
	for _, res := range resources {
		ns := res.Namespace()
		if ns == "" {
			ns = "(cluster)"
		}
		if _, found := uniqNames[ns]; !found {
			names = append(names, ns)
			uniqNames[ns] = struct{}{}
		}
	}
	sort.Strings(names)
	return names
}

func (r *DeployRequest) newResources(
	prep ctlapp.Preparation, labeledResources *ctlres.LabeledResources,
	resourceFilter ctlres.ResourceFilter) ([]ctlres.Resource, ctlconf.Conf, []string, error) {

	var newResources []ctlres.Resource

	inlineResources, err := ctlres.NewFileResource(ctlres.NewBytesSource([]byte(r.yaml))).Resources()
	if err != nil {
		return nil, ctlconf.Conf{}, nil, err
	}

	newResources = append(newResources, inlineResources...)

	for _, file := range r.files {
		fileRs, err := ctlres.NewFileResources(file)
		if err != nil {
			log.Fatalf("Error create file resources: %s", err)
		}

		for _, fileRes := range fileRs {
			resources, err := fileRes.Resources()
			if err != nil {
				log.Fatalf("Error create getting resources from file: %s", err)
			}

			newResources = append(newResources, resources...)
		}
	}

	newResources, conf, err := ctlconf.NewConfFromResourcesWithDefaults(newResources)
	if err != nil {
		return nil, ctlconf.Conf{}, nil, err
	}

	newResources, err = prep.PrepareResources(newResources)
	if err != nil {
		return nil, ctlconf.Conf{}, nil, err
	}

	err = labeledResources.Prepare(newResources, conf.OwnershipLabelMods(),
		conf.LabelScopingMods(), conf.AdditionalLabels())
	if err != nil {
		return nil, ctlconf.Conf{}, nil, err
	}

	// Grab ns names before resource filtering is applied
	nsNames := r.nsNames(newResources)

	return resourceFilter.Apply(newResources), conf, nsNames, nil
}
