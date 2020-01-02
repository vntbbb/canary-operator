package controller

import (
	"github.com/vntbbb/canary-operator/pkg/controller/canarydeploy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, canarydeploy.Add)
}
