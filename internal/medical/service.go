package medical

import (
	"context"
	"database/sql"
	"errors"

	"clinic-backend/internal/audit"
	"clinic-backend/internal/inventory"
	"clinic-backend/internal/procedurecatalog"
	"github.com/google/uuid"
)

var (
	ErrRecordNotFound = errors.New("medical record not found")
)

type MedicalService struct {
	repo         *MedicalRepository
	audit        *audit.AuditService
	inventoryRepo inventory.Repository
	procRepo      procedurecatalog.Repository
}

func NewMedicalService(repo *MedicalRepository, audit *audit.AuditService, invRepo inventory.Repository, procRepo procedurecatalog.Repository) *MedicalService {
	return &MedicalService{repo: repo, audit: audit, inventoryRepo: invRepo, procRepo: procRepo}
}

func (s *MedicalService) CreateRecord(tenantID, doctorID, patientID uuid.UUID, req CreateMedicalRecordRequest) (*MedicalRecordResponse, error) {
	rec := &MedicalRecord{
		TenantID:      tenantID,
		PatientID:     patientID,
		DoctorID:      doctorID,
		AppointmentID: req.AppointmentID,
		Diagnosis:     req.Diagnosis,
		Notes:         req.Notes,
	}

	var vitals []*MedicalVital
	var meds []*MedicalMedication

	err := s.repo.RunInTransaction(func(tx *sql.Tx) error {
		if err := s.repo.CreateRecord(tx, rec); err != nil {
			return err
		}

		for _, vReq := range req.Vitals {
			v := &MedicalVital{
				MedicalRecordID: rec.ID,
				Type:            vReq.Type,
				Value:           vReq.Value,
				Unit:            vReq.Unit,
			}
			if err := s.repo.CreateVital(tx, v); err != nil {
				return err
			}
			vitals = append(vitals, v)
		}

		for _, mReq := range req.Medications {
			m := &MedicalMedication{
				MedicalRecordID: rec.ID,
				Name:            mReq.Name,
				Dosage:          mReq.Dosage,
				Frequency:       mReq.Frequency,
				Duration:        mReq.Duration,
				Notes:           mReq.Notes,
			}
			if err := s.repo.CreateMedication(tx, m); err != nil {
				return err
			}
			meds = append(meds, m)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	s.audit.LogAction(tenantID, doctorID, "CREATE_MEDICAL_RECORD", "medical_records", rec.ID, rec)

	return &MedicalRecordResponse{
		Record:      rec,
		Vitals:      vitals,
		Medications: meds,
	}, nil
}

func (s *MedicalService) GetRecord(tenantID, recordID uuid.UUID) (*MedicalRecordResponse, error) {
	rec, err := s.repo.GetRecordByID(tenantID, recordID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	vitals, err := s.repo.GetVitalsByRecordID(recordID)
	if err != nil {
		return nil, err
	}

	meds, err := s.repo.GetMedicationsByRecordID(recordID)
	if err != nil {
		return nil, err
	}

	procs, err := s.repo.GetProceduresByRecordID(recordID)
	if err != nil {
		return nil, err
	}

	return &MedicalRecordResponse{
		Record:      rec,
		Vitals:      vitals,
		Medications: meds,
		Procedures:  procs,
	}, nil
}

func (s *MedicalService) ListRecordsByPatient(tenantID, patientID uuid.UUID) ([]*MedicalRecordResponse, error) {
	records, err := s.repo.GetRecordsByPatientID(tenantID, patientID)
	if err != nil {
		return nil, err
	}

	responses := make([]*MedicalRecordResponse, 0, len(records))
	for _, rec := range records {
		vitals, _ := s.repo.GetVitalsByRecordID(rec.ID)
		meds, _ := s.repo.GetMedicationsByRecordID(rec.ID)
		procs, _ := s.repo.GetProceduresByRecordID(rec.ID)

		responses = append(responses, &MedicalRecordResponse{
			Record:      rec,
			Vitals:      vitals,
			Medications: meds,
			Procedures:  procs,
		})
	}

	return responses, nil
}

func (s *MedicalService) UpdateRecord(tenantID, doctorID, recordID uuid.UUID, req UpdateMedicalRecordRequest) (*MedicalRecordResponse, error) {
	rec, err := s.repo.GetRecordByID(tenantID, recordID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	if req.Diagnosis != nil {
		rec.Diagnosis = *req.Diagnosis
	}
	if req.Notes != nil {
		rec.Notes = *req.Notes
	}
	rec.AppointmentID = req.AppointmentID

	var newVitals []*MedicalVital
	var newMeds []*MedicalMedication

	err = s.repo.RunInTransaction(func(tx *sql.Tx) error {
		if err := s.repo.UpdateRecord(tx, tenantID, rec); err != nil {
			return err
		}

		if err := s.repo.DeleteVitalsByRecordID(tx, recordID); err != nil {
			return err
		}
		for _, vReq := range req.Vitals {
			v := &MedicalVital{
				MedicalRecordID: rec.ID,
				Type:            vReq.Type,
				Value:           vReq.Value,
				Unit:            vReq.Unit,
			}
			if err := s.repo.CreateVital(tx, v); err != nil {
				return err
			}
			newVitals = append(newVitals, v)
		}

		if err := s.repo.DeleteMedicationsByRecordID(tx, recordID); err != nil {
			return err
		}
		for _, mReq := range req.Medications {
			m := &MedicalMedication{
				MedicalRecordID: rec.ID,
				Name:            mReq.Name,
				Dosage:          mReq.Dosage,
				Frequency:       mReq.Frequency,
				Duration:        mReq.Duration,
				Notes:           mReq.Notes,
			}
			if err := s.repo.CreateMedication(tx, m); err != nil {
				return err
			}
			newMeds = append(newMeds, m)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.audit.LogAction(tenantID, doctorID, "UPDATE_MEDICAL_RECORD", "medical_records", rec.ID, rec)

	return &MedicalRecordResponse{
		Record:      rec,
		Vitals:      newVitals,
		Medications: newMeds,
	}, nil
}

func (s *MedicalService) DeleteRecord(tenantID, doctorID, recordID uuid.UUID) error {
	err := s.repo.DeleteRecord(tenantID, recordID)
	if err == nil {
		s.audit.LogAction(tenantID, doctorID, "DELETE_MEDICAL_RECORD", "medical_records", recordID, nil)
	}
	return err
}

func (s *MedicalService) AddProcedureToRecord(ctx context.Context, tenantID, doctorID, recordID uuid.UUID, req AddProcedureReq) (*MedicalRecordProcedure, error) {
	// Verify record exists and belongs to tenant
	_, err := s.repo.GetRecordByID(tenantID, recordID)
	if err != nil {
		return nil, err
	}

	// Fetch procedure catalog template
	procTemplate, err := s.procRepo.GetProcedureByID(ctx, tenantID, req.ProcedureCatalogID)
	if err != nil {
		return nil, errors.New("procedure catalog template not found")
	}

	procRecord := &MedicalRecordProcedure{
		TenantID:           tenantID,
		MedicalRecordID:    recordID,
		ProcedureCatalogID: req.ProcedureCatalogID,
		PerformedBy:        &doctorID,
		Notes:              req.Notes,
	}

	err = s.repo.RunInTransaction(func(tx *sql.Tx) error {
		// 1. Create the procedure record
		if err := s.repo.CreateProcedureRecord(tx, procRecord); err != nil {
			return err
		}

		// 2. Deduct inventory items
		for _, item := range procTemplate.Items {
			reason := "Procedure: " + procTemplate.Name
			// We use AdjustStockTx directly from the inventory repo to participate in the same TX if we wanted to
			// Wait, the inventory repo's AdjustStockTx requires *sql.Tx. We CAN pass it if we share the DB connection.
			// Since we use `s.repo.RunInTransaction` which uses the same db.Begin(), we can pass `tx`.
			err := s.inventoryRepo.AdjustStockTx(ctx, tx, tenantID, item.InventoryItemID, "out", item.Quantity, &reason, nil, &recordID, &doctorID)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.audit.LogAction(tenantID, doctorID, "ADD_PROCEDURE_TO_RECORD", "medical_record_procedures", procRecord.ID, procRecord)

	return procRecord, nil
}
