{{/*
Expand the name of the chart.
*/}}
{{- define "modelsrv-k8s-sensor.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "modelsrv-k8s-sensor.fullname" -}}
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
Labels applied to all resources
*/}}
{{- define "modelsrv-k8s-sensor.labels" -}}
helm.sh/chart: {{ include "modelsrv-k8s-sensor.chart" . }}
{{ include "modelsrv-k8s-sensor.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "modelsrv-k8s-sensor.selectorLabels" -}}
app.kubernetes.io/name: {{ include "modelsrv-k8s-sensor.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "modelsrv-k8s-sensor.chart" -}}
{{- printf "%s-%s" .Chart.Name (.Chart.Version | replace "+" "_") }}
{{- end }}

{{- define "modelsrv-k8s-sensor.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "modelsrv-k8s-sensor.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "modelsrv-k8s-sensor.operatorSystemName" -}}
{{- default (printf "%s-operator" (include "modelsrv-k8s-sensor.fullname" .)) .Values.operator.systemName }}
{{- end }}

{{- define "modelsrv-k8s-sensor.operatorApiName" -}}
{{- default (printf "%s-api" (include "modelsrv-k8s-sensor.fullname" .)) .Values.operator.apiName }}
{{- end }}

{{- define "modelsrv-k8s-sensor.operatorComponentName" -}}
{{- default (printf "%s-component" (include "modelsrv-k8s-sensor.fullname" .)) .Values.operator.componentName }}
{{- end }}

{{- define "modelsrv-k8s-sensor.operatorVersion" -}}
{{- default .Chart.AppVersion .Values.operator.systemVersion }}
{{- end }}
