module github.com/niallthomson/terraform-provider-k14s

go 1.13

replace github.com/aryann/difflib => github.com/k14s/difflib v0.0.0-20200108171459-b101e55e0592

replace go.starlark.net => github.com/k14s/starlark-go v0.0.0-20200207164905-fd8842955e4e // ytt branch

require (
	github.com/cppforlife/go-cli-ui v0.0.0-20200108172221-38b12a2f8675
	github.com/google/uuid v1.1.1
	github.com/hashicorp/hcl/v2 v2.3.0 // indirect
	github.com/hashicorp/terraform-config-inspect v0.0.0-20191212124732-c6ae6269b9d7 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.8.0
	github.com/k14s/kapp v0.22.0
	github.com/k14s/terraform-provider-k14s v0.4.0
	github.com/k14s/ytt v0.26.0
	k8s.io/client-go v8.0.0+incompatible
)
