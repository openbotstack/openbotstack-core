A previous execution plan failed. Generate a revised execution plan to complete the task.
/no_think

Original Plan Steps:
{{range $i, $step := .OriginalSteps}}{{add $i 1}}. [{{$step.Type}}] {{$step.Name}}{{if $step.Args}} args={{$step.Args}}{{end}}
{{end}}
Failed Step: [{{.FailedStepType}}] {{.FailedStepName}}
Failure Reason: {{.ErrorMessage}}
Trigger: {{.Trigger}}
{{- if .PreviousResults}}

Completed Step Results (Do NOT repeat these steps):
{{range $name, $result := .PreviousResults}}
- {{$name}}: {{$result}}{{end}}
{{- end}}
{{- if .Personality}}

Personality: {{.Personality}}
{{- end}}
{{- if .Instructions}}

Specific Instructions:
{{.Instructions}}
{{- end}}
{{- if .TurnResults}}

Previous Turn Results:
{{range .TurnResults}}
- [{{.StepType}}: {{.StepName}}] {{if .Success}}OK{{else}}FAILED{{end}}{{if .Summary}}: {{.Summary}}{{end}}{{if .Error}}: {{.Error}}{{end}}
{{- end}}
{{- end}}
{{- if .MemoryContext}}

Relevant Memory Context:
{{range .MemoryContext}}
- {{.}}
{{- end}}
{{- end}}

Available skills/tools:
{{.Skills}}

<user_request>
{{.UserRequest}}
</user_request>

Respond with a JSON object containing the revised execution plan. Do not include any other text or reasoning.
Format:
{
  "assistant_id": "...",
  "steps": [
    {"type": "tool", "name": "builtin.now", "arguments": {}},
    {"type": "skill", "name": "summarize", "arguments": {"text": "..."}},
    {"type": "llm", "name": "respond", "arguments": {"prompt": "..."}}
  ]
}

IMPORTANT:
- Generate only the steps needed to complete the REMAINING work.
- Do NOT repeat already completed steps.
- Use "type": "tool" for mcp.* and builtin.* tools. Use "type": "skill" for skills. Use "type": "llm" for direct LLM responses.
- The "name" field of a step MUST be the exact tool/skill identifier (e.g. "builtin.resource_read"). Do NOT invent labels like "pdf_content" — the harness dispatches and keys results by this exact name.
- Reference earlier step outputs with {{`{{step_name}}`}} or {{`{{step_name.field}}`}}, where step_name is the EXACT value of that step's "name" field (e.g. {{`{{builtin.resource_read.content}}`}} or short {{`{{resource_read.content}}`}}). Never use an invented label.
/no_think
