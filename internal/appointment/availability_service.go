package appointment

import (
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
)

type Slot struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	IsAvailable bool      `json:"is_available"`
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

func (s *AvailabilityService) GetAvailableSlots(tenantID uuid.UUID, doctorID *uuid.UUID, dateFrom, dateTo time.Time) ([]DoctorAvailabilityResponse, error) {
	var docIDs []uuid.UUID
	if doctorID != nil {
		docIDs = []uuid.UUID{*doctorID}
	}
	
	appts, err := s.repo.GetAppointmentsInRange(tenantID, docIDs, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	apptsByDoctor := make(map[uuid.UUID][]Appointment)
	for _, a := range appts {
		apptsByDoctor[a.DoctorID] = append(apptsByDoctor[a.DoctorID], a)
	}

	slotDuration := 30 * time.Minute
	slotsByDoctor := make(map[uuid.UUID][]Slot)
	
	// Normalize the date range to UTC midnight boundaries
	d := time.Date(dateFrom.Year(), dateFrom.Month(), dateFrom.Day(), 0, 0, 0, 0, time.UTC)
	endD := time.Date(dateTo.Year(), dateTo.Month(), dateTo.Day(), 23, 59, 59, 0, time.UTC)
	nowUTC := time.Now().UTC()

	for ; d.Before(endD); d = d.AddDate(0, 0, 1) {
		dayOfWeek := int(d.Weekday())
		
		avails, err := s.repo.GetDoctorAvailabilities(tenantID, docIDs, dayOfWeek)
		if err != nil {
			return nil, err
		}

		for _, avail := range avails {
			st, err := time.Parse("15:04:05", avail.StartTime)
			if err != nil { continue }
			et, err := time.Parse("15:04:05", avail.EndTime)
			if err != nil { continue }

			dayStart := time.Date(d.Year(), d.Month(), d.Day(), st.Hour(), st.Minute(), st.Second(), 0, time.UTC)
			dayEnd   := time.Date(d.Year(), d.Month(), d.Day(), et.Hour(), et.Minute(), et.Second(), 0, time.UTC)

			for cur := dayStart; cur.Before(dayEnd); cur = cur.Add(slotDuration) {
				curEnd := cur.Add(slotDuration)
				
				overlap := false
				for _, appt := range apptsByDoctor[avail.DoctorID] {
					// Normalise to UTC to match booking validation
					apptStart := appt.StartTime.UTC()
					apptEnd   := appt.EndTime.UTC()
					if cur.Before(apptEnd) && curEnd.After(apptStart) {
						overlap = true
						break
					}
				}
				
				isAvail := !overlap && cur.After(nowUTC)
				slotsByDoctor[avail.DoctorID] = append(slotsByDoctor[avail.DoctorID], Slot{
					StartTime:   cur,
					EndTime:     curEnd,
					IsAvailable: isAvail,
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
			DoctorName: "", // Optional: could be populated by a separate doctor service call if needed.
			Slots:      slots,
		})
	}

	return results, nil
}

func (s *AvailabilityService) NextAvailableSlot(tenantID uuid.UUID, doctorID *uuid.UUID) (*Slot, error) {
	now := time.Now()
	// Look ahead up to 14 days
	maxDate := now.AddDate(0, 0, 14) 
	
	slotsResp, err := s.GetAvailableSlots(tenantID, doctorID, now, maxDate)
	if err != nil {
		return nil, err
	}
	
	var earliest *Slot
	for _, docSlots := range slotsResp {
		for _, slot := range docSlots.Slots {
			if slot.IsAvailable {
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
