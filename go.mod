module github.com/bakito/k8s-event-logger-operator

go 1.16

require (
	github.com/bakito/operator-utils v1.3.2
	github.com/fatih/structs v1.1.0
	github.com/go-logr/logr v1.2.0
	github.com/go-playground/locales v0.14.0
	github.com/go-playground/universal-translator v0.18.0
	github.com/go-playground/validator/v10 v10.9.0
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo/v2 v2.0.0-rc2
	github.com/onsi/gomega v1.17.0
	k8s.io/api v0.23.1
	k8s.io/apimachinery v0.23.1
	k8s.io/client-go v0.23.1
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/controller-runtime v0.10.3
)
