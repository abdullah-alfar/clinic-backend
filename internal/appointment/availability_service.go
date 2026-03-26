package appointment

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

type Slot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

type DoctorAvailabilityResponse struct {
	DoctorID   uuid.UUID `json:"doctor_id"`
	DoctorName string    `json:"doctor_name"`
	Slots      []Slot    `json:"slots"`
}

type AvailabilityService struct {
	repo AppointmentRepository
}

func NewAvailabilityService(repo AppointmentRepository) *AvailabilityService {
	return &AvailabilityService{repo: repo}
}

func (s *AvailabilityService) GetAvailableSlots(tenantID uuid.UUID, doctorID *uuid.UUID, dateFrom, dateTo time.Time, timezone string) ([]DoctorAvailabilityResponse, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	var docIDs []uuid.UUID
	if doctorID != nil {
		docIDs = []uuid.UUID{*doctorID}
	}

	// Normalize the date range to Clinic-local midnight boundaries
	d := time.Date(dateFrom.Year(), dateFrom.Month(), dateFrom.Day(), 0, 0, 0, 0, loc)
	endD := time.Date(dateTo.Year(), dateTo.Month(), dateTo.Day(), 23, 59, 59, 0, loc)
	nowLocal := time.Now().In(loc)

	appts, err := s.repo.GetAppointmentsInRange(tenantID, docIDs, d, endD)
	if err != nil {
		return nil, err
	}
	fmt.Printf("DEBUG: Fetched %d appointments for range %v to %v\n", len(appts), d, endD)

	apptsByDoctor := make(map[uuid.UUID][]Appointment)
	for _, a := range appts {
		apptsByDoctor[a.DoctorID] = append(apptsByDoctor[a.DoctorID], a)
	}

	slotDuration := 30 * time.Minute
	slotsByDoctor := make(map[uuid.UUID][]Slot)
	
	// Reset d for the loop
	dLoop := d
	for ; dLoop.Before(endD); dLoop = dLoop.AddDate(0, 0, 1) {
		dayOfWeek := int(dLoop.Weekday())
		
		avails, err := s.repo.GetDoctorAvailabilities(tenantID, docIDs, dayOfWeek)
		if err != nil {
			return nil, err
		}

		for _, avail := range avails {
			st, err := time.Parse("15:04:05", avail.StartTime)
			if err != nil { continue }
			et, err := time.Parse("15:04:05", avail.EndTime)
			if err != nil { continue }

			dayStart := time.Date(dLoop.Year(), dLoop.Month(), dLoop.Day(), st.Hour(), st.Minute(), st.Second(), 0, loc)
			dayEnd   := time.Date(dLoop.Year(), dLoop.Month(), dLoop.Day(), et.Hour(), et.Minute(), et.Second(), 0, loc)

			for cur := dayStart; cur.Before(dayEnd); cur = cur.Add(slotDuration) {
				curEnd := cur.Add(slotDuration)
				
				overlap := false
				for _, appt := range apptsByDoctor[avail.DoctorID] {
					if cur.Before(appt.EndTime) && curEnd.After(appt.StartTime) {
						fmt.Printf("DEBUG: Found overlap for slot %v with appointment %v - %v\n", cur, appt.StartTime, appt.EndTime)
						overlap = true
						break
					}
				}
				
				status := "available"
				if overlap {
					status = "booked"
				} else if !cur.After(nowLocal) {
					status = "unavailable"
				}

				slotsByDoctor[avail.DoctorID] = append(slotsByDoctor[avail.DoctorID], Slot{
					StartTime: cur,
					EndTime:   curEnd,
					Status:    status,
				})
			}
		}
	}

	var results []DoctorAvailabilityResponse
	for dID, slots := range slotsByDoctor {
		sort.Slice(slots, func(i, j int) bool {
			return slots[i].StartTime.Before(slots[j].StartTime)
		})
		results = append(results, DoctorAvailabilityResponse{
			DoctorID:   dID,
			DoctorName: "", // Optional
			Slots:      slots,
		})
	}

	return results, nil
}

func (s *AvailabilityService) NextAvailableSlot(tenantID uuid.UUID, doctorID *uuid.UUID) (*Slot, error) {
	now := time.Now()
	// Look ahead up to 14 days
	maxDate := now.AddDate(0, 0, 14) 
	
	tz, _ := s.repo.GetTenantTimezone(tenantID)
	slotsResp, err := s.GetAvailableSlots(tenantID, doctorID, now, maxDate, tz)

	if err != nil {
		return nil, err
	}
	
	var earliest *Slot
	for _, docSlots := range slotsResp {
		for _, slot := range docSlots.Slots {
			if slot.Status == "available" {
				if earliest == nil || slot.StartTime.Before(earliest.StartTime) {
					earliestCopy := slot
					earliest = &earliestCopy
				}
			}
		}
	}
	
	if earliest == nil {
		return nil, errors.New("no available slots found")
	}
	
	return earliest, nil
}
