{{/*
Loom Helm Chart — Template Helpers
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "loom.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "loom.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart label (name + version)
*/}}
{{- define "loom.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to all resources
*/}}
{{- define "loom.labels" -}}
helm.sh/chart: {{ include "loom.chart" . }}
{{ include "loom.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: loom
{{- end }}

{{/*
Selector labels (used in matchLabels + pod labels)
*/}}
{{- define "loom.selectorLabels" -}}
app.kubernetes.io/name: {{ include "loom.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component-specific selector labels
*/}}
{{- define "loom.componentLabels" -}}
{{ include "loom.selectorLabels" . }}
app.kubernetes.io/component: {{ . }}
{{- end }}

{{/*
ServiceAccount name for the loom control plane
*/}}
{{- define "loom.serviceAccountName" -}}
{{- if .Values.loom.serviceAccount.create }}
{{- default (include "loom.fullname" .) .Values.loom.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.loom.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Resolve effective image pull policy (component overrides global)
*/}}
{{- define "loom.imagePullPolicy" -}}
{{- $component := index . 0 -}}
{{- $global := index . 1 -}}
{{- if $component }}{{ $component }}{{- else }}{{ $global }}{{- end }}
{{- end }}

{{/*
Full image reference for loom control plane
*/}}
{{- define "loom.image" -}}
{{- $reg := .Values.global.imageRegistry -}}
{{- $repo := .Values.loom.image.repository -}}
{{- $tag := .Values.loom.image.tag | default .Chart.AppVersion -}}
{{- if $reg }}{{ $reg }}/{{ $repo }}:{{ $tag }}{{- else }}{{ $repo }}:{{ $tag }}{{- end }}
{{- end }}

{{/*
Full image reference for connectors-service
*/}}
{{- define "loom.connectorsImage" -}}
{{- $reg := .Values.global.imageRegistry -}}
{{- $repo := .Values.connectorsService.image.repository -}}
{{- $tag := .Values.connectorsService.image.tag | default .Chart.AppVersion -}}
{{- if $reg }}{{ $reg }}/{{ $repo }}:{{ $tag }}{{- else }}{{ $repo }}:{{ $tag }}{{- end }}
{{- end }}

{{/*
Full image reference for project agents
*/}}
{{- define "loom.agentImage" -}}
{{- $reg := .Values.global.imageRegistry -}}
{{- $repo := .Values.agents.image.repository -}}
{{- $tag := .Values.agents.image.tag | default .Chart.AppVersion -}}
{{- if $reg }}{{ $reg }}/{{ $repo }}:{{ $tag }}{{- else }}{{ $repo }}:{{ $tag }}{{- end }}
{{- end }}

{{/*
Loom postgresql secret name
*/}}
{{- define "loom.postgresqlSecretName" -}}
{{- if .Values.postgresql.auth.existingSecret -}}
{{ .Values.postgresql.auth.existingSecret }}
{{- else -}}
{{ include "loom.fullname" . }}-postgresql-secret
{{- end }}
{{- end }}

{{/*
Loom admin secret name
*/}}
{{- define "loom.secretName" -}}
{{- if .Values.loom.existingSecret -}}
{{ .Values.loom.existingSecret }}
{{- else -}}
{{ include "loom.fullname" . }}-secret
{{- end }}
{{- end }}

{{/*
Resolve effective storageClass (component overrides global)
*/}}
{{- define "loom.storageClass" -}}
{{- $sc := . -}}
{{- if $sc }}{{ $sc }}{{- else }}""{{- end }}
{{- end }}

{{/*
NATS service URL inside the cluster
*/}}
{{- define "loom.natsURL" -}}
nats://{{ .Release.Name }}-nats:4222
{{- end }}

{{/*
PostgreSQL host via pgbouncer (when enabled) or directly
*/}}
{{- define "loom.dbHost" -}}
{{- if .Values.pgbouncer.enabled -}}
{{ include "loom.fullname" . }}-pgbouncer
{{- else -}}
{{ include "loom.fullname" . }}-postgresql
{{- end }}
{{- end }}

{{/*
Temporal host
*/}}
{{- define "loom.temporalHost" -}}
{{- if .Values.temporal.enabled -}}
{{ .Release.Name }}-temporal-frontend:7233
{{- else -}}
{{ .Values.loom.config.temporalHost }}
{{- end }}
{{- end }}
