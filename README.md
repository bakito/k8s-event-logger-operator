[![operator docker Repository on Quay](https://quay.io/repository/bakito/k8s-event-logger-operator/status "operator docker Repository on Quay")](https://quay.io/repository/bakito/k8s-event-logger-operator) operator  
[![logger docker Repository on Quay](https://quay.io/repository/bakito/k8s-event-logger/status "logger docker Repository on Quay")](https://quay.io/repository/bakito/k8s-event-logger) logger  
[![Go Report Card](https://goreportcard.com/badge/github.com/bakito/k8s-event-logger-operator)](https://goreportcard.com/report/github.com/bakito/k8s-event-logger-operator) 
[![GitHub Release](https://img.shields.io/github/release/bakito/k8s-event-logger-operator.svg?style=flat)](https://github.com/bakito/k8s-event-logger-operator/releases)

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
    - name: DeploymentConfig # the kind of the event source to be loggeed
      eventTypes: # optional
       - Noramal
       - Warning
      matchingPatterns: # optional - regexp pattern to match event messages
       - .*
      skipOnMatch: false # optional - skip events where messages match the pattern. Default false


  eventTypes: # optional - define the event types to log. If no types are defined, all events are logged
    - Noramal
    - Warning

  labels: # optional - additional labels for the pod
    name: value

  annotations: # optional - additional annotations for the pod
    name: value

  scrapeMetrics: false # optional att prometeus scrape metrics annotation to the pod. Default false

  namespace: "ns" # optional - the namespace to lsten the events on. Default the current namespace

  serviceAccount: "sa" # optional - if a custom ServiceAccount should be used for the pod. Default ServiceAccount is automatically created
```
