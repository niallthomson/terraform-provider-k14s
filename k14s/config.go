package k14s

import (
	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type Config struct {
	DepsFactory cmdcore.DepsFactory
}
