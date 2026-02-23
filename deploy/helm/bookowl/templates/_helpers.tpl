{{/*
Expand the name of the chart.
*/}}
{{- define "bookowl.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "bookowl.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "bookowl.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "bookowl.labels" -}}
helm.sh/chart: {{ include "bookowl.chart" . }}
{{ include "bookowl.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "bookowl.selectorLabels" -}}
app.kubernetes.io/name: {{ include "bookowl.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
API selector labels.
*/}}
{{- define "bookowl.apiSelectorLabels" -}}
{{ include "bookowl.selectorLabels" . }}
app.kubernetes.io/component: api
{{- end }}

{{/*
Web selector labels.
*/}}
{{- define "bookowl.webSelectorLabels" -}}
{{ include "bookowl.selectorLabels" . }}
app.kubernetes.io/component: web
{{- end }}

{{/*
Service account name.
*/}}
{{- define "bookowl.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "bookowl.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
API image reference.
*/}}
{{- define "bookowl.apiImage" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Web image reference.
*/}}
{{- define "bookowl.webImage" -}}
{{- $tag := default .Chart.AppVersion .Values.web.image.tag -}}
{{- printf "%s:%s" .Values.web.image.repository $tag }}
{{- end }}
