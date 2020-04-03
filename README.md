# terraform-provider-k14sx

Experimental Terraform provider for the [k14s](https://github.com/k14s) toolchain. See [here](https://github.com/k14s/terraform-provider-k14s) for the official one.

This implementation aims to improve on several aspects of the official version:
- No requirement for the binaries locally
- Works with flows where the target Kubernetes cluster is built during TF execution

*WARNING:* This code base is undergoing active development and breaking changes should be expected

TODO list
- Handle more complex kubeconfig provider configuration
- Handle kapp configuration drift
- Expose more flags and configuration options as necessary

## Installation

Grab prebuilt binaries from the [Releases page](https://github.com/niallthomson/terraform-provider-k14sx/releases).

Once you have downloaded `terraform-provider-k14sx-binaries.tgz`, install it for Terraform to find it:

```bash
mkdir -p ~/.terraform.d/plugins
tar xzvf ~/Downloads/terraform-provider-k14sx-binaries.tgz -C ~/.terraform.d/plugins/
```

You do not need to have `kapp` or `ytt` installed locally.

## Example

The following simple example creates an Nginx deployment manifest with a patched in namespace of `mynamespace`, which is then deployed via `kapp`:

```
provider "k14sx" {

}

data "k14sx_ytt" "content" {
  config_yaml = [<<EOF
#@ load("@ytt:data", "data")
#@ load("@ytt:overlay", "overlay")

#@overlay/match by=overlay.subset({"kind": "Deployment", "metadata":{"name": "nginx-deployment"}})
---
metadata:
  #@overlay/match missing_ok=True
  namespace: mynamespace
EOF
]

  files = [
    "https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/controllers/nginx-deployment.yaml"
  ]
}

resource "k14sx_kapp" "app" {
  app = "example"
  namespace = "default"

  config_yaml = data.k14sx_ytt.content.result
}
```


## Building Locally

First clone this repository.

If you just want to build the code:

```
go build .
```

If you want to build the code and install the plugin on your local machine:

```
./install.sh
```