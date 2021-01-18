module github.com/bakito/k8s-event-logger-operator

go 1.15

require (
	github.com/bakito/operator-utils v1.2.0
	github.com/fatih/structs v1.1.0
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/go-playground/locales v0.13.0
	github.com/go-playground/universal-translator v0.17.0
	github.com/go-playground/validator/v10 v10.4.1
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.4
	github.com/google/uuid v1.1.4
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.19.4
	sigs.k8s.io/controller-runtime v0.6.4
)
