# terraform-provider-k14s

Experimental Terraform provider for the [k14s](https://github.com/k14s) toolchain. See [here](https://github.com/k14s/terraform-provider-k14s) for the official one. It aims to be compatible with the official version (but is not feature complete) so its easy to switch between them.

This implementation aims to improve on several aspects of the official version:
- No requirement for the binaries locally
- Works with flows where the target Kubernetes cluster is built during TF execution

TODO list
- Handle more complex kubeconfig provider configuration
- Handle kapp configuration drift
