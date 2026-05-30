# Screen List — Functional Spec

Each screen is described as: **who's there, what they're trying to do, what success looks like, what failure looks like, edge cases.** No layout descriptions, no current-implementation references — design however serves these jobs best.

{{- range .ScreenSections }}

## {{ .Letter }}. {{ .Title }}{{ if .ThreadHint }} ({{ .ThreadHint }}){{ end }}

{{- range .Screens }}

### {{ .ID }} — {{ .Title }}

**Who:** {{ .Who }}
**JTBD:** {{ .JTBD }}
**Success:** {{ .Success }}
{{- if .Failure }}
**Failure:** {{ .Failure }}
{{- end }}
{{- if .EdgeCases }}
**Edge cases:** {{ .EdgeCases }}
{{- end }}
{{- if .Constraints }}
**Constraints:** {{ .Constraints }}
{{- end }}
{{- end }}
{{- end }}

## Cross-cutting

{{- range .CrossCutting }}
- **{{ .Heading }}** — {{ .Body }}
{{- else }}
- **Dark mode** — every screen has a dark variant.
- **Reduced motion** — every animated screen has a reduced-motion path.
- **{{.OfflineOrConnectivityHeading}}** — {{.OfflineOrConnectivityBody}}
- **{{.MultiplicityHeading}}** — every screen that shows {{.MultiplicityKind}}-specific data must work for {{.MultiplicityCases}} without redesign.
{{- end }}
