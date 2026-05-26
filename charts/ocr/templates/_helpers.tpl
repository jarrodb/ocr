{{- define "ocr.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ocr.labels" -}}
app.kubernetes.io/name: {{ include "ocr.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "ocr.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ocr.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
