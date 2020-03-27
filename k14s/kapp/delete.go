package kapp

import (
	"time"

	"github.com/cppforlife/go-cli-ui/ui"
	ctlapp "github.com/k14s/kapp/pkg/kapp/app"
	ctlcap "github.com/k14s/kapp/pkg/kapp/clusterapply"
	"github.com/k14s/kapp/pkg/kapp/cmd/app"
	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
	ctldiff "github.com/k14s/kapp/pkg/kapp/diff"
	ctldgraph "github.com/k14s/kapp/pkg/kapp/diffgraph"
	ctlres "github.com/k14s/kapp/pkg/kapp/resources"
	util "github.com/niallthomson/terraform-provider-k14s/k14s/util"
)

type DeleteRequest struct {
	configFactory cmdcore.ConfigFactory
	name          string
	namespace     string
}

func NewDeleteRequest(configFactory cmdcore.ConfigFactory, name string, namespace string) *DeleteRequest {
	return &DeleteRequest{
		configFactory: configFactory,
		name:          name,
		namespace:     namespace,
	}
}

func (r *DeleteRequest) Execute() error {
	failingAPIServicesPolicy := &app.FailingAPIServicesPolicy{}

	logger := util.NewStdOutLogger()

	ui := &util.LoggingUI{}
	defer ui.Flush()

	depsFactory := util.NewDepsFactoryImpl(r.configFactory)

	app, supportObjs, err := app.AppFactory(depsFactory, app.AppFlags{
		Name: r.name,
		NamespaceFlags: cmdcore.NamespaceFlags{
			Name: r.namespace,
		},
	}, app.ResourceTypesFlags{}, logger)
	if err != nil {
		return err
	}

	exists, err := app.Exists()
	if err != nil {
		return err
	}

	if !exists {
		//o.ui.PrintLinef("App '%s' (namespace: %s) does not exist",
		//	app.Name(), o.AppFlags.NamespaceFlags.Name)
		return nil
	}

	usedGVs, err := app.UsedGVs()
	if err != nil {
		return err
	}

	failingAPIServicesPolicy.MarkRequiredGVs(usedGVs)

	existingResources, fullyDeleteApp, err := r.existingResources(app, supportObjs)
	if err != nil {
		return err
	}

	clusterChangeSet, clusterChangesGraph, _, err :=
		r.calculateAndPresentChanges(existingResources, supportObjs, ui)
	if err != nil {
		return err
	}

	touch := ctlapp.Touch{App: app, Description: "delete", IgnoreSuccessErr: true}

	err = touch.Do(func() error {
		err := clusterChangeSet.Apply(clusterChangesGraph)
		if err != nil {
			return err
		}
		if fullyDeleteApp {
			return app.Delete()
		}
		return nil
	})
	if err != nil {
		return err
	}

	/*if o.ApplyFlags.ExitStatus {
		return DeployApplyExitStatus{hasNoChanges}
	}*/
	return nil
}

func (r *DeleteRequest) existingResources(app ctlapp.App,
	supportObjs app.AppFactorySupportObjs) ([]ctlres.Resource, bool, error) {

	labelSelector, err := app.LabelSelector()
	if err != nil {
		return nil, false, err
	}

	existingResources, err := supportObjs.IdentifiedResources.List(labelSelector)
	if err != nil {
		return nil, false, err
	}

	resourceFilter := ctlres.ResourceFilter{}

	fullyDeleteApp := true
	applicableExistingResources := resourceFilter.Apply(existingResources)

	if len(applicableExistingResources) != len(existingResources) {
		fullyDeleteApp = false
		/*ui.PrintLinef("App '%s' (namespace: %s) will not be fully deleted "+
		"because some resources are excluded by filters",
		app.Name(), o.AppFlags.NamespaceFlags.Name)*/
	}

	existingResources = applicableExistingResources

	r.changeIgnored(existingResources)

	return existingResources, fullyDeleteApp, nil
}

func (r *DeleteRequest) calculateAndPresentChanges(existingResources []ctlres.Resource,
	supportObjs app.AppFactorySupportObjs, ui ui.UI) (ctlcap.ClusterChangeSet, *ctldgraph.ChangeGraph, bool, error) {

	var clusterChangeSet ctlcap.ClusterChangeSet

	changeSetOpts := ctldiff.ChangeSetOpts{
		AgainstLastApplied: true,
	}

	// whats the actual defaults for all these?
	clusterChangeOpts := ctlcap.ClusterChangeOpts{
		ApplyIgnored: false,
		Wait:         true,
		WaitIgnored:  true,

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

	{ // Figure out changes for X existing resources -> 0 new resources
		changeFactory := ctldiff.NewChangeFactory(nil, nil)
		changeSetFactory := ctldiff.NewChangeSetFactory(changeSetOpts, changeFactory)

		changes, err := changeSetFactory.New(existingResources, nil).Calculate()
		if err != nil {
			return ctlcap.ClusterChangeSet{}, nil, false, err
		}

		{ // Build cluster changes based on diff changes
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
	}

	clusterChanges, clusterChangesGraph, err := clusterChangeSet.Calculate()
	if err != nil {
		return ctlcap.ClusterChangeSet{}, nil, false, err
	}

	return clusterChangeSet, clusterChangesGraph, (len(clusterChanges) == 0), nil
}

const (
	ownedForDeletionAnnKey = "kapp.k14s.io/owned-for-deletion" // valid values: ''
)

func (r *DeleteRequest) changeIgnored(resources []ctlres.Resource) {
	// Good example for use of this annotation is PVCs created by StatefulSet
	// (PVCs do not get deleted when StatefulSet is deleted:
	// https://github.com/k14s/kapp/issues/36)
	for _, res := range resources {
		if _, found := res.Annotations()[ownedForDeletionAnnKey]; found {
			res.MarkTransient(false)
		}
	}
}
