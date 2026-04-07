package ops_intelligence

import (
	"strings"
)

type Analyzer interface {
	Analyze(records []MedicalRecordData, invoices []InvoiceData) MissingRevenue
}

type defaultAnalyzer struct {
	keywords []string
}

func NewAnalyzer() Analyzer {
	return &defaultAnalyzer{
		keywords: []string{"stitches", "x-ray", "blood test", "biopsy", "injection", "ultrasound"},
	}
}

func (a *defaultAnalyzer) Analyze(records []MedicalRecordData, invoices []InvoiceData) MissingRevenue {
	missing := []string{}
	foundInNotes := make(map[string]bool)

	for _, m := range records {
		for _, kw := range a.keywords {
			if strings.Contains(strings.ToLower(m.Notes), kw) || strings.Contains(strings.ToLower(m.Diagnosis), kw) {
				foundInNotes[kw] = true
			}
		}
	}

	// Simple check to see if those keywords are already mentioned in any invoice amounts/items
	// Since we don't have invoice items in this simple version, we'll just check if there's any invoice.
	// In a more complex version, we'd check against line items.
	
	if len(invoices) == 0 {
		for kw := range foundInNotes {
			missing = append(missing, kw)
		}
	}

	return MissingRevenue{
		MissingServices: missing,
	}
}
