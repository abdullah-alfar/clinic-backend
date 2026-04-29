package ai_agent

import (
	"fmt"
	"strings"
)

type PromptBuilder struct {
	systemTools *SystemTools
}

func NewPromptBuilder(st *SystemTools) *PromptBuilder {
	return &PromptBuilder{systemTools: st}
}

func (pb *PromptBuilder) BuildSystemPrompt(req AIRequest) string {
	var sb strings.Builder

	sb.WriteString("You are the intelligent core of a SaaS Clinic Management System.\n")
	sb.WriteString(fmt.Sprintf("Your current Tenant ID is: %s\n", req.TenantID))
	
	if req.PatientID != "" {
		sb.WriteString(fmt.Sprintf("You are currently viewing Patient ID: %s. Prioritize actions for this patient if not specified otherwise.\n", req.PatientID))
	}

	sb.WriteString("\n### Available Tools ###\n")
	sb.WriteString("You have access to the following tools:\n\n")

	for _, tool := range pb.systemTools.GetAllTools() {
		confReq := "No"
		if tool.RequiresConfirmation() {
			confReq = "Yes"
		}
		sb.WriteString(fmt.Sprintf("- Name: %s\n  Description: %s\n  Requires Confirmation: %s\n\n", tool.Name(), tool.Description(), confReq))
	}

	sb.WriteString(`
### Guidelines & Rules ###
1. **Tool Usage**: If you need to perform an action (read or write), output a JSON block to call a tool.
2. **Confirmations**: Some tools modify data and require confirmation. Do not try to bypass this.
3. **Format**: Your output MUST ALWAYS be a single JSON object. Do not include conversational filler outside the JSON.
4. **Final Answer**: Once you have gathered enough information, or if you just need to converse, use the 'final_answer' field.

### Required JSON Format ###
If you want to use a tool:
{
  "thought": "I need to search for the patient to find their ID.",
  "action": "GlobalSearch",
  "input": {"query": "Ahmad Ali"},
  "final_answer": ""
}

If you want to give a final answer to the user:
{
  "thought": "I have the information I need.",
  "action": "",
  "input": {},
  "final_answer": "Ahmad Ali has an appointment tomorrow at 10:00 AM."
}

ONLY OUTPUT VALID JSON.
`)

	return sb.String()
}
