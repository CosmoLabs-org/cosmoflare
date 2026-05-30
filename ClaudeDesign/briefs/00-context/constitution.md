# {{.ProjectName}} Constitution

## Vision

{{.Vision}}

## Target users

{{- range .PrimaryPersonas }}

**{{ if .Tier }}{{ .Tier }} persona{{ else }}Persona{{ end }} — {{ .Name }}{{ if .Role }}, {{ .Role }}{{ end }}{{ if .Age }} ({{ .Age }}){{ end }}.** {{ .Description }}
{{- end }}

{{- if .SharedTraits }}

All personas share these traits:
{{- range $i, $t := .SharedTraits }}
{{ add $i 1 }}. {{ $t }}
{{- end }}
{{- end }}

## Jobs-to-be-done

In priority order. The flagship is **JTBD-1**.

| ID | Job | Trigger | Done when |
|---|---|---|---|
{{- range .JTBDs }}
| **{{ .ID }}** | "{{ .Job }}" | {{ .Trigger }} | {{ .DoneWhen }} |
{{- end }}

## Emotional context

{{.EmotionalContext}}

The app must be:

{{- range .EmotionalQualities }}
- **{{ .Heading }}** {{ .Body }}
{{- end }}

## Non-negotiable principles

{{- range $i, $p := .NonNegotiablePrinciples }}
{{ add $i 1 }}. **{{ $p.Heading }}** {{ $p.Body }}
{{- end }}

## What {{.ProjectName}} is not

{{- range .NotList }}
- {{ . }}
{{- end }}

If a feature serves a different job than the JTBDs above, it doesn't belong.
