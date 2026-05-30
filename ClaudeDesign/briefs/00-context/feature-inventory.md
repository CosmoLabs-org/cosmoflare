# Feature Inventory — User Outcomes

Each entry is framed as a **user outcome**, not an implementation. If you find yourself thinking "that's hard to render," propose the rendering you'd actually want; the implementation will follow.

{{- range $i, $f := .Features }}

## {{ add $i 1 }}. {{ $f.Name }}{{ if $f.IsFlagship }} — the flagship{{ end }}

**The user wants:** {{ $f.UserWants }}
{{- if $f.MathOrLogic }}

**{{ $f.MathOrLogicLabel }}, concisely:** {{ $f.MathOrLogic }}
{{- end }}

**Surfaces this shows up on:** {{ $f.Surfaces }}.

**Edge cases the design must handle:**
{{- range $f.EdgeCases }}
- {{ . }}
{{- end }}
{{- end }}

---

## What's deliberately not in this list

{{- range .NotInList }}
- {{ . }}
{{- end }}

If a design instinct points toward any of these, redirect — they don't serve the JTBDs in `constitution.md`.

## Ranking the inventory by visibility

If you have to prioritize what gets the most polish, this is the order:

{{- range $i, $r := .VisibilityRanking }}
{{ add $i 1 }}. **{{ $r.Heading }}** — {{ $r.Body }}
{{- end }}
