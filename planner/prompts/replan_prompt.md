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
- Only generate steps using the available skills/tools listed above. Never invent skill or tool names.
- Reference earlier step outputs with {{`{{step_name}}`}} in arguments.
/no_think
