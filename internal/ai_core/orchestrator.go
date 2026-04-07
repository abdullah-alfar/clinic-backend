package ai_core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"clinic-backend/internal/ai"
)

type Orchestrator struct {
	provider      ai.Provider
	memory        MemoryManager
	systemTools   *SystemTools
	promptBuilder *PromptBuilder
	maxIterations int
}

func NewOrchestrator(provider ai.Provider, memory MemoryManager, st *SystemTools, pb *PromptBuilder) *Orchestrator {
	return &Orchestrator{
		provider:      provider,
		memory:        memory,
		systemTools:   st,
		promptBuilder: pb,
		maxIterations: 5, // Maximum ReAct cycles to prevent infinite loops
	}
}

// Orchestrate handles the ReAct loop until a final answer is determined.
func (o *Orchestrator) Orchestrate(ctx context.Context, req AIRequest) (AIResponse, error) {
	// Add user's latest input
	o.memory.AddObservation(req.SessionID, ReactObservation{
		Role:    "user",
		Content: req.Input,
	})

	for i := 0; i < o.maxIterations; i++ {
		// 1. Build Full Prompt including Context, Identity, History
		fullPrompt := o.buildExecutionPrompt(req)

		// 2. Call the LLM
		llmOut, err := o.provider.Generate(ctx, fullPrompt)
		if err != nil {
			return AIResponse{}, fmt.Errorf("llm generation error: %w", err)
		}

		// Clean potential markdown blocks
		llmOut = cleanJSONString(llmOut)

		// Record assistant thought
		o.memory.AddObservation(req.SessionID, ReactObservation{
			Role:    "assistant",
			Content: llmOut,
		})

		// 3. Parse JSON Intent
		var result ReactResult
		err = json.Unmarshal([]byte(llmOut), &result)
		if err != nil {
			// Parsing failed - AI didn't follow the JSON directive. Pass raw text back.
			o.memory.AddObservation(req.SessionID, ReactObservation{
				Role:    "tool",
				Content: "Error: You must output ONLY valid JSON using the format specified earlier.",
			})
			continue
		}

		// 4. Was it a final answer? Exit condition.
		if result.FinalAnswer != "" {
			return AIResponse{
				Message: result.FinalAnswer,
			}, nil
		}

		// 5. It wants to call an Action!
		if result.Action != "" {
			tool, exists := o.systemTools.GetTool(result.Action)
			if !exists {
				// Tool doesn't exist
				o.memory.AddObservation(req.SessionID, ReactObservation{
					Role:    "tool",
					Content: fmt.Sprintf("Error: Tool '%s' does not exist.", result.Action),
				})
				continue
			}

			// Execute the Tool
			toolOut, err := tool.Execute(ctx, req.TenantID, req.PatientID, result.Input)
			if err != nil {
				o.memory.AddObservation(req.SessionID, ReactObservation{
					Role:    "tool",
					Content: fmt.Sprintf("Error executing tool: %v", err),
				})
				continue
			}

			// Pass successful tool output back into history
			o.memory.AddObservation(req.SessionID, ReactObservation{
				Role:    "tool",
				Content: fmt.Sprintf("Tool %s executed successfully. Output:\n%s", result.Action, toolOut),
			})

			// End loop step, system will automatically ask LLM again via next loop iteration
			continue
		}

		// Unreachable unless JSON parsed correctly but was empty
		o.memory.AddObservation(req.SessionID, ReactObservation{
			Role:    "tool",
			Content: "Error: JSON parsed but no action or final_answer found. Follow the format.",
		})
	}

	return AIResponse{
		Message: "I'm sorry, I needed too many steps to figure that out and timed out. Could you try asking more specifically?",
	}, nil
}

func (o *Orchestrator) buildExecutionPrompt(req AIRequest) string {
	var sb strings.Builder

	// Write static instruction block
	sb.WriteString(o.promptBuilder.BuildSystemPrompt(req))

	sb.WriteString("### Conversation History & Observations:\n")

	history := o.memory.GetHistory(req.SessionID)
	for _, obs := range history {
		switch obs.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("\nUSER: %s\n", obs.Content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("\nASSISTANT: %s\n", obs.Content))
		case "tool":
			sb.WriteString(fmt.Sprintf("\nTOOL_OBSERVATION: %s\n", obs.Content))
		}
	}

	sb.WriteString("\n\nNow, generate your response as raw JSON.\n")
	return sb.String()
}

func cleanJSONString(in string) string {
	in = strings.TrimSpace(in)
	in = strings.TrimPrefix(in, "```json")
	in = strings.TrimPrefix(in, "```")
	in = strings.TrimSuffix(in, "```")
	return strings.TrimSpace(in)
}
