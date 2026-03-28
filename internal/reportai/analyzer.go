package reportai

import (
	"context"
	"encoding/json"
	"time"
)

type AIProvider interface {
	AnalyzeReport(ctx context.Context, fileURL string, mimeType string) (summary string, structuredData json.RawMessage, err error)
}

type MockAIProvider struct{}

func NewMockAIProvider() *MockAIProvider {
	return &MockAIProvider{}
}

func (m *MockAIProvider) AnalyzeReport(ctx context.Context, fileURL string, mimeType string) (string, json.RawMessage, error) {
	// Simulate an AI processing delay for realistic UX
	time.Sleep(2 * time.Second)

	summary := "This is a simulated AI analysis of the uploaded report. The report appears to show normal baseline values for most requested panels. There are no critical flags indicating urgent medical intervention. Patient is advised to maintain current lifestyle."

	structData := map[string]interface{}{
		"abnormal_results": false,
		"key_metrics": []string{
			"Blood Pressure: 120/80 (Normal)",
			"Heart Rate: 72 bpm (Normal)",
		},
		"recommendations": []string{
			"Continue current diet",
			"Suggest regular exercise 3x a week",
		},
		"disclaimer": "AI assistance only. Not medical advice.",
	}

	bytes, _ := json.Marshal(structData)
	return summary, bytes, nil
}
