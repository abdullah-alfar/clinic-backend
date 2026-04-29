package ai_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"clinic-backend/internal/ai"
)

type MemoryManager interface {
	AddObservation(sessionID string, obs ReactObservation)
	GetHistory(sessionID string) []ReactObservation
	ClearHistory(sessionID string)
}

type Orchestrator struct {
	provider      ai.Provider
	memory        MemoryManager
	systemTools   *SystemTools
	promptBuilder *PromptBuilder
	confirmStore  ConfirmationStore
	maxIterations int
}

func NewOrchestrator(
	provider ai.Provider,
	memory MemoryManager,
	st *SystemTools,
	pb *PromptBuilder,
	cs ConfirmationStore,
) *Orchestrator {
	return &Orchestrator{
		provider:      provider,
		memory:        memory,
		systemTools:   st,
		promptBuilder: pb,
		confirmStore:  cs,
		maxIterations: 5,
	}
}

func (o *Orchestrator) Orchestrate(ctx context.Context, req AIRequest) (AIResponse, error) {
	o.memory.AddObservation(req.SessionID, ReactObservation{
		Role:    "user",
		Content: req.Input,
	})

	for i := 0; i < o.maxIterations; i++ {
		fullPrompt := o.buildExecutionPrompt(req)

		llmOut, err := o.provider.Generate(ctx, fullPrompt)
		if err != nil {
			return AIResponse{}, fmt.Errorf("llm generation error: %w", err)
		}

		llmOut = cleanJSONString(llmOut)

		o.memory.AddObservation(req.SessionID, ReactObservation{
			Role:    "assistant",
			Content: llmOut,
		})

		var result ReactResult
		err = json.Unmarshal([]byte(llmOut), &result)
		if err != nil {
			o.memory.AddObservation(req.SessionID, ReactObservation{
				Role:    "tool",
				Content: "Error: You must output ONLY valid JSON. Parsing failed.",
			})
			continue
		}

		if result.FinalAnswer != "" {
			return AIResponse{
				Message: result.FinalAnswer,
			}, nil
		}

		if result.Action != "" {
			tool, exists := o.systemTools.GetTool(result.Action)
			if !exists {
				o.memory.AddObservation(req.SessionID, ReactObservation{
					Role:    "tool",
					Content: fmt.Sprintf("Error: Tool '%s' does not exist.", result.Action),
				})
				continue
			}

			toolOut, err := tool.Execute(ctx, req.TenantID, req.PatientID, result.Input)
			if err != nil {
				o.memory.AddObservation(req.SessionID, ReactObservation{
					Role:    "tool",
					Content: fmt.Sprintf("Error executing tool %s: %v", result.Action, err),
				})
				continue
			}

			if tool.RequiresConfirmation() {
				// Tool requires confirmation, stop and ask the user.
				action := PendingAction{
					Token:   GenerateToken(),
					Type:    tool.Name(),
					Payload: result.Input,
					Summary: toolOut, // the Execute method for write tools should return a summary
				}

				if err := o.confirmStore.Save(action); err != nil {
					return AIResponse{}, fmt.Errorf("failed to save confirmation: %w", err)
				}

				o.memory.AddObservation(req.SessionID, ReactObservation{
					Role:    "tool",
					Content: fmt.Sprintf("Action %s is pending user confirmation.", result.Action),
				})

				return AIResponse{
					Message:              toolOut,
					RequiresConfirmation: true,
					Action:               &action,
				}, nil
			}

			o.memory.AddObservation(req.SessionID, ReactObservation{
				Role:    "tool",
				Content: fmt.Sprintf("Tool %s executed successfully. Output:\n%s", result.Action, toolOut),
			})

			continue
		}

		o.memory.AddObservation(req.SessionID, ReactObservation{
			Role:    "tool",
			Content: "Error: JSON parsed but no action or final_answer found.",
		})
	}

	return AIResponse{
		Message: "I reached the maximum number of steps trying to process this request. Please try rephrasing.",
	}, nil
}

func (o *Orchestrator) buildExecutionPrompt(req AIRequest) string {
	var sb strings.Builder

	sb.WriteString(o.promptBuilder.BuildSystemPrompt(req))
	sb.WriteString("\n### Conversation History & Observations:\n")

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
