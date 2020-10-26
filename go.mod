module github.com/bakito/k8s-event-logger-operator

go 1.14

require (
	github.com/fatih/structs v1.1.0
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.2
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.6.3
)
