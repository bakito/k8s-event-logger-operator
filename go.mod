module github.com/bakito/k8s-event-logger-operator

go 1.15

require (
	github.com/bakito/operator-utils v1.2.0
	github.com/fatih/structs v1.1.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/go-playground/locales v0.13.0
	github.com/go-playground/universal-translator v0.17.0
	github.com/go-playground/validator/v10 v10.4.1
	github.com/golang/mock v1.5.0
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.2.0
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.11.0
	golang.org/x/sys v0.0.0-20210304152209-afaa3650a925 // indirect
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.6.4
)
