module github.com/bakito/k8s-event-logger-operator

go 1.14

require (
	github.com/fatih/structs v1.1.0
	// fix untli 0.2.1 is released https://github.com/go-logr/logr/issues/22
	github.com/go-logr/logr v0.2.1-0.20200730175230-ee2de8da5be6
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.1
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	sigs.k8s.io/controller-runtime v0.6.2
)
