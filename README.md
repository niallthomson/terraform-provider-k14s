# terraform-provider-k14s

Experimental Terraform provider for the k14s toolchain. See here for the official one.

This implementation aims to improve on several aspects of the official version:
- No requirement for the binaries locally
- Works with flows where the target Kubernetes cluster is built during TF execution

TODO list
- Handle more complex kubeconfig provider configuration
- Handle kapp configuration drift