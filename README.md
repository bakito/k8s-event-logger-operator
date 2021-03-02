[![Build Status](https://travis-ci.com/bakito/k8s-event-logger-operator.svg?branch=master)](https://travis-ci.com/bakito/k8s-event-logger-operator) [![Go Report Card](https://goreportcard.com/badge/github.com/bakito/k8s-event-logger-operator)](https://goreportcard.com/report/github.com/bakito/k8s-event-logger-operator)

[![GitHub Release](https://img.shields.io/github/release/bakito/k8s-event-logger-operator.svg?style=flat)](https://github.com/bakito/k8s-event-logger-operator/releases) [![Coverage Status](https://coveralls.io/repos/github/bakito/k8s-event-logger-operator/badge.svg?branch=master)](https://coveralls.io/github/bakito/k8s-event-logger-operator?branch=master)

# k8s event logger operator

This operator creates a logging pod that logs corev1.Event information as structured json log.
The crd allows to configure the events to be logged.

## Installation

### Operator
The operator is insalled with helm.

```bash
helm upgrade --install eventlogger ./helm/
```

### Custom Resource Definition (CRD)

```yaml
apiVersion: eventlogger.bakito.ch/v1
kind: EventLogger
metadata:
  name: example-eventlogger
spec:
  kinds:
    - name: DeploymentConfig # the kind of the event source to be logged
      apiGroup: apps.openshift.io # optional
      eventTypes: # optional
       - Normal
       - Warning
      reasons: # optional
       - DeploymentCreated
       - ReplicationControllerScaled
      matchingPatterns: # optional - regexp pattern to match event messages
       - .*
      skipOnMatch: false # optional - skip events where messages match the pattern. Default false


  eventTypes: # optional - define the event types to log. If no types are defined, all events are logged
    - Normal
    - Warning

  labels: # optional - additional labels for the pod
    name: value

  annotations: # optional - additional annotations for the pod
    name: value

  scrapeMetrics: false # optional att prometheus scrape metrics annotation to the pod. Default false

  namespace: "ns" # optional - the namespace to listen the events on. Default the current namespace

  nodeSelector: # optional - a node selector for the logging pod.
    key: value

  serviceAccount: "sa" # optional - if a custom ServiceAccount should be used for the pod. Default ServiceAccount is automatically created
  
  logFields: # optional - map if custom log field names. Key then log field name / Value: the reflection fields to the value within the struct corev1.Event https://github.com/kubernetes/api/blob/master/core/v1/types.go
    - name: name
      path:
        - InvolvedObject
        - Name 
    - name: kind
      path:
        - InvolvedObject
        - Kind
    - name: type
      path:
        - Type
    - name: some-static-value
      value: ""
```
