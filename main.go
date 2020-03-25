package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/niallthomson/terraform-provider-k14s/k14s"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: k14s.Provider})
}
