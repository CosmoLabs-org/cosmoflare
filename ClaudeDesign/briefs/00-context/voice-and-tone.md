# Voice & Tone

## Brand personality, in one sentence

{{.ProjectName}} is **{{.BrandPersonality}}**.

## Adjective ladder

Use these as the personality dial. The app is:

{{- range .BrandVoice.Adjectives }}
- **{{ .Adjective }}** — {{ .Definition }}
{{- end }}

## What it never sounds like

| Anti-pattern | Example | Why it's wrong |
|---|---|---|
{{- range .BrandVoice.AntiPatterns }}
| {{ .Pattern }} | "{{ .Example }}" | {{ .Why }} |
{{- end }}

## Microcopy patterns

{{- range .BrandVoice.MicrocopyPatterns }}

### {{ .Section }}

{{- range .Examples }}
- **{{ .Label }}:** *"{{ .Copy }}"*
{{- if .Note }}
  ({{ .Note }})
{{- end }}
{{- end }}

{{- if .PatternNote }}

{{ .PatternNote }}
{{- end }}
{{- end }}

## Microinteractions

- Buttons say verbs: *{{.VerbExamples}}.*
- No "Submit." No "OK." No "Click here."
- "Cancel" is allowed. "Dismiss" is allowed. "Done" is allowed (when there's no save).

## Numbers and dates

- Dates: *"12 Sep 2026"* in long form, *"12/09/26"* never (ambiguous DMY/MDY).
- Day counts and quantities: be specific. *"3 items"*, *"14 days remaining"*.
- {{.DomainNomenclatureRule}}
- Time: relative (*"2 hours ago"*) for recent events; absolute for scheduled or historical entries.

## Localization-ready (future)

The app is English-first now but designed for translation later. Avoid:
- Idiom (*"a stone's throw away"*)
- US-centric date formats
- Cultural references

Prefer:
- Direct factual language
- ISO date components
- Universal iconography

## Tone rotation by surface

The voice is consistent, but the *intensity* varies by surface:

| Surface | Tone notes |
|---|---|
{{- range .BrandVoice.ToneRotation }}
| {{ .Surface }} | {{ .Notes }} |
{{- end }}

## What to do when in doubt

{{.VoiceRubric}}
