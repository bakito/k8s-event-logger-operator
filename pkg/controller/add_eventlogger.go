package controller

import (
	"github.com/bakito/k8s-event-logger-operator/pkg/controller/eventlogger"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, eventlogger.Add)
}
