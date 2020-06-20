module github.com/bakito/k8s-event-logger-operator

go 1.13

require (
	github.com/fatih/structs v1.1.0
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.3
	github.com/golang/mock v1.3.1
	github.com/google/go-cmp v0.4.0
	github.com/operator-framework/operator-sdk v0.18.1
	github.com/spf13/pflag v1.0.5
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200121204235-bf4fb3bd569c
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
)
