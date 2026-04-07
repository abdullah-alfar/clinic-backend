package ai_core

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PromptBuilder struct {
	systemTools *SystemTools
}

func NewPromptBuilder(systemTools *SystemTools) *PromptBuilder {
	return &PromptBuilder{systemTools: systemTools}
}

// BuildSystemPrompt generates the static instructions + the dynamically injected tools
func (b *PromptBuilder) BuildSystemPrompt(req AIRequest) string {
	var sb strings.Builder

	// 1. Identity & Persona
	sb.WriteString("You are the core intelligence of the Clinic Management SaaS. " +
		"You are a highly capable orchestrator who decides what tools to call based on the user's intent. " +
		"You MUST strictly follow the JSON output format required.\n\n")

	// 2. Context Injection
	sb.WriteString("### Current Context:\n")
	sb.WriteString(fmt.Sprintf("- Tenant ID: %s\n", req.TenantID))
	sb.WriteString(fmt.Sprintf("- Session ID: %s\n", req.SessionID))
	if req.PatientID != nil {
		sb.WriteString(fmt.Sprintf("- Acting on behalf of Patient: %s\n", req.PatientID))
	} else if req.UserID != nil {
		sb.WriteString(fmt.Sprintf("- Acting on behalf of User (Staff): %s\n", req.UserID))
	}
	sb.WriteString(fmt.Sprintf("- Request Source: %s\n", req.Source))

	if len(req.Context) > 0 {
		ctxBytes, _ := json.MarshalIndent(req.Context, "", "  ")
		sb.WriteString(fmt.Sprintf("- Additional States:\n%s\n", string(ctxBytes)))
	}
	sb.WriteString("\n")

	// 3. Tool Injection
	sb.WriteString("### Available Tools:\n")
	sb.WriteString("You have access to the following server-side functions. You cannot invent new ones.\n")
	tools := b.systemTools.AllTools()
	toolsData := make([]map[string]interface{}, 0, len(tools))
	
	for _, t := range tools {
		toolsData = append(toolsData, map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"schema":      t.Schema(),
		})
	}
	
	toolsBytes, _ := json.MarshalIndent(toolsData, "", "  ")
	sb.WriteString(string(toolsBytes) + "\n\n")

	// 4. Output Rules (The ReAct Loop)
	sb.WriteString("### Output Format Rules (STRICT):\n")
	sb.WriteString("You MUST ALWAYS output a raw valid JSON object. No markdown wrappers like ```json. \n")
	sb.WriteString("If you need to call a tool, output exactly this format:\n")
	sb.WriteString(`{
  "action": "<tool_name>",
  "input": { // match the schema exactly }
}` + "\n")
	sb.WriteString("The system will intercept this, run the tool, and pass the results back to you as 'tool_response'.\n\n")

	sb.WriteString("If you have enough information to form the final answer, output exactly this format:\n")
	sb.WriteString(`{
  "final_answer": "Your human-readable response here."
}` + "\n")

	sb.WriteString("\nIf there is an error or you don't know the answer, use `final_answer` to naturally ask the user for clarification.\n")

	return sb.String()
}
