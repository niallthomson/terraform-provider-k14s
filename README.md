# terraform-provider-k14sx

Experimental Terraform provider for the [k14s](https://github.com/k14s) toolchain. See [here](https://github.com/k14s/terraform-provider-k14s) for the official one.

This implementation aims to improve on several aspects of the official version:
- No requirement for the binaries locally
- Works with flows where the target Kubernetes cluster is built during TF execution

TODO list
- Handle more complex kubeconfig provider configuration
- Handle kapp configuration drift
- Expose more flags and configuration options as necessary