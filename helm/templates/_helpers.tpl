{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "k8s-event-logger-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "k8s-event-logger-operator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "k8s-event-logger-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "k8s-event-logger-operator.labels" -}}
helm.sh/chart: {{ include "k8s-event-logger-operator.chart" . }}
helm.sh/namespace: {{ .Release.Namespace }}
{{ include "k8s-event-logger-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "k8s-event-logger-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8s-event-logger-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "k8s-event-logger-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "k8s-event-logger-operator.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Create the name of the role to use
*/}}
{{- define "k8s-event-logger-operator.roleName" -}}
{{- if .Values.rbac.create -}}
    {{ default (include "k8s-event-logger-operator.fullname" .) .Values.rbac.roleName }}
{{- else -}}
    {{ default "default" .Values.rbac.roleName }}
{{- end -}}
{{- end -}}

{{/*
Get the webhook cert secret name
*/}}
{{- define "k8s-event-logger-operator.webhookCertSecretName" -}}
{{- default (printf "%s-webhook" (include "k8s-event-logger-operator.fullname" .))  .Values.webhook.certsSecret.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
