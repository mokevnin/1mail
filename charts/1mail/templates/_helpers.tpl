{{- define "1mail.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "1mail.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{- define "1mail.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "1mail.image" -}}
{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}
{{- end -}}

{{/* Shared container environment: plain settings (secrets come from envFrom). */}}
{{- define "1mail.env" -}}
- name: APP_ENV
  value: production
- name: PORT
  value: {{ .Values.containerPort | quote }}
- name: AUTO_MIGRATE
  value: "false"
{{- with .Values.appUrl }}
- name: APP_URL
  value: {{ . | quote }}
{{- end }}
{{- if .Values.metrics.enabled }}
- name: METRICS_ADDR
  value: {{ printf "0.0.0.0:%d" (int .Values.metrics.port) | quote }}
{{- end }}
{{- with .Values.extraEnv }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "1mail.envFrom" -}}
- secretRef:
    name: {{ required "existingSecret is required" .Values.existingSecret }}
{{- end -}}
