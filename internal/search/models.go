package search

type EntityType string

const (
	EntityPatient EntityType = "patient"
	EntityDoctor  EntityType = "doctor"
	EntityReport  EntityType = "report"
	EntityMemory  EntityType = "memory"
)
