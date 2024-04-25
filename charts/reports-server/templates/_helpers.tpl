{{/*
Expand the name of the chart.
*/}}
{{- define "reports-server.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "reports-server.fullname" -}}
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
{{- define "reports-server.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "reports-server.labels" -}}
helm.sh/chart: {{ include "reports-server.chart" . }}
{{ include "reports-server.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "reports-server.selectorLabels" -}}
app.kubernetes.io/name: {{ include "reports-server.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "reports-server.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "reports-server.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Database config is injected into the environment, if a secret ref is set. Otherwise, Helm values are used directly.
*/}}
{{- define "reports-server.dbHost" -}}
{{- if .Values.config.db.secretName }}
{{- printf "%s" "$(DB_HOST)" }}
{{- else }}
{{- default (printf "%s-postgresql.%s" $.Release.Name $.Release.Namespace ) .Values.config.db.host }}
{{- end }}
{{- end }}

{{- define "reports-server.dbName" -}}
{{- if .Values.config.db.secretName }}
{{- printf "%s" "$(DB_DATABASE)" }}
{{- else }}
{{- .Values.config.db.name }}
{{- end }}
{{- end }}

{{- define "reports-server.dbUser" -}}
{{- if .Values.config.db.secretName }}
{{- printf "%s" "$(DB_USER)" }}
{{- else }}
{{- .Values.config.db.user }}
{{- end }}
{{- end }}

{{- define "reports-server.dbPassword" -}}
{{- if .Values.config.db.secretName }}
{{- printf "%s" "$(DB_PASSWORD)" }}
{{- else }}
{{- .Values.config.db.password }}
{{- end }}
{{- end }}

