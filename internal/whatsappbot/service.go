package whatsappbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"clinic-backend/internal/appointment"
	"clinic-backend/internal/availability"
	"clinic-backend/internal/doctor"
	"clinic-backend/internal/whatsapp"
	"clinic-backend/internal/ai_core"
	"github.com/google/uuid"
)

type BotService struct {
	repo         BotRepository
	sender       whatsapp.WhatsAppSender
	appointments *appointment.AppointmentService
	availability *availability.AvailabilityService
	doctors      *doctor.DoctorService
	aiCore       ai_core.AIService
}

func NewBotService(
	repo BotRepository,
	sender whatsapp.WhatsAppSender,
	appts *appointment.AppointmentService,
	avail *availability.AvailabilityService,
	docs *doctor.DoctorService,
	aiCore ai_core.AIService,
) *BotService {
	return &BotService{
		repo:         repo,
		sender:       sender,
		appointments: appts,
		availability: avail,
		doctors:      docs,
		aiCore:       aiCore,
	}
}

// ProcessInbound handles an incoming message from the webhook.
func (s *BotService) ProcessInbound(ctx context.Context, tenantID uuid.UUID, msg InboundMessage) error {
	phone, err := whatsapp.NormalizePhone(msg.From)
	if err != nil || strings.TrimSpace(phone) == "" {
		phone = strings.TrimSpace(msg.From)
		phone = strings.TrimPrefix(phone, "whatsapp:")
	}
	// 1. Load or Create Session
	session, err := s.repo.GetSession(ctx, tenantID, phone)
	if err != nil {
		// Create new
		session = &BotSession{
			ID:          uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: phone,
			CurrentFlow: "menu",
			CurrentStep: "start",
			State:       make(StateData),
		}

		// Attempt to link patient
		patID, _ := s.repo.FindPatientByPhone(ctx, tenantID, phone)
		session.PatientID = patID
		s.repo.UpsertSession(ctx, session)
	}

	// 2. Log Inbound
	_ = s.repo.LogMessage(ctx, tenantID, session.PatientID, phone, "inbound", "text", msg.Body, msg.ProviderMsgID)

	// 3. Instead of simple intent parsed string matching, we pass this to ai_core
	aiReq := ai_core.AIRequest{
		SessionID: fmt.Sprintf("wa-%s-%s", tenantID.String(), phone),
		TenantID:  tenantID,
		PatientID: session.PatientID,
		Input:     msg.Body,
		Source:    "whatsapp",
		Context: map[string]interface{}{
			"current_flow": session.CurrentFlow,
			"current_step": session.CurrentStep,
		},
	}

	aiResp, err := s.aiCore.Process(ctx, aiReq)
	if err != nil {
		// Fallback to legacy flow if AI fails or is disabled
		intent := ParseIntent(msg.Body)
		if session.CurrentFlow == "menu" || session.CurrentStep == "start" {
			if intent != IntentUnknown && intent != IntentSelection && intent != IntentConfirmation {
				session.CurrentFlow = intent
				session.CurrentStep = "start"
				session.State.Clear()
				s.repo.UpsertSession(ctx, session)
			}
		}
		reply := s.handleFlow(ctx, session, msg.Body)
		if reply.Body != "" {
			s.sendReply(ctx, session, reply)
		}
		return nil
	}

	// 4. Act on AI Response
	if aiResp.Message != "" {
		s.sendReply(ctx, session, OutboundReply{Body: aiResp.Message})
	}
	
	// If the AI took conclusive action ending a flow it might return Action: reset_flow
	if aiResp.Action == "reset_flow" {
		session.CurrentFlow = "menu"
		session.CurrentStep = "start"
		s.repo.UpsertSession(ctx, session)
	}

	return nil
}

func (s *BotService) handleFlow(ctx context.Context, session *BotSession, text string) OutboundReply {
	if session.PatientID == nil {
		session.CurrentFlow = "menu"
		session.CurrentStep = "start"
		s.repo.UpsertSession(ctx, session)
		return OutboundReply{Body: "Hello! We don't have this phone number registered to any patient. Please contact the clinic receptionist to update your profile. 👋"}
	}

	switch session.CurrentFlow {
	case IntentBookAppointment:
		return s.flowBook(ctx, session, text)
	case IntentCancelAppointment:
		return s.flowCancel(ctx, session, text)
	case IntentViewNext:
		return s.flowNext(ctx, session, text)
	case IntentSendReport:
		return s.flowReport(ctx, session, text)
	case IntentHelp:
		fallthrough
	default:
		return s.flowMenu(ctx, session)
	}
}

func (s *BotService) flowMenu(ctx context.Context, session *BotSession) OutboundReply {
	session.CurrentFlow = "menu"
	session.CurrentStep = "start"
	s.repo.UpsertSession(ctx, session)

	msg := "Welcome to your Clinic Assistant! 🏥\n\nHow can I help you today? Please reply with one of the following:\n\n1️⃣ Book Appointment\n2️⃣ Cancel Appointment\n3️⃣ View Next Appointment\n4️⃣ Get my Medical Reports"
	return OutboundReply{Body: msg}
}

func (s *BotService) flowBook(ctx context.Context, session *BotSession, text string) OutboundReply {
	switch session.CurrentStep {
	case "start":
		session.CurrentStep = "awaiting_date"
		s.repo.UpsertSession(ctx, session)
		return OutboundReply{Body: "Sure! What date would you like to book for? (e.g., Tomorrow, or 2024-05-15) 📅"}

	case "awaiting_date":
		// Naive date parsing for MVP
		targetDate, err := s.parseDateInput(text)
		if err != nil {
			return OutboundReply{Body: "I couldn't understand that date. Please try something like 'Tomorrow' or '2024-05-15'. 🧐"}
		}

		// Use last doctor or first available
		doctorID, err := s.appointments.GetLastDoctorIDForPatient(session.TenantID, *session.PatientID)
		if doctorID == nil {
			// Fallback: Get first doctor from List
			docs, _ := s.doctors.List(session.TenantID)
			if len(docs) > 0 {
				doctorID = &docs[0].ID
			}
		}

		if doctorID == nil {
			return OutboundReply{Body: "I'm sorry, I couldn't find any available doctors at the moment. Please contact the clinic. 🏥"}
		}

		// Get slots
		slots, err := s.availability.GetAvailableSlots(ctx, session.TenantID, *doctorID, availability.SlotQueryParams{
			DateFrom: targetDate,
			DateTo:   targetDate,
		})

		if err != nil || len(slots.Slots) == 0 {
			return OutboundReply{Body: fmt.Sprintf("I'm sorry, there are no available slots for %s. Please try another date. 🗓️", targetDate.Format("2006-01-02"))}
		}

		// Filter for available slots
		var availSlots []availability.SlotDTO
		for _, sl := range slots.Slots {
			if sl.Status == "available" {
				availSlots = append(availSlots, sl)
				if len(availSlots) >= 5 {
					break
				}
			}
		}

		if len(availSlots) == 0 {
			return OutboundReply{Body: "It looks like we're fully booked for that day. Try a different date? 📅"}
		}

		// Save slots in state for selection
		session.State.Set("doctor_id", doctorID.String())
		session.State.Set("temp_slots", availSlots)
		session.CurrentStep = "awaiting_slot"
		s.repo.UpsertSession(ctx, session)

		msg := fmt.Sprintf("Great! Here are available slots for %s. Please reply with a number (1-%d):\n\n", targetDate.Format("Mon, Jan 2"), len(availSlots))
		for i, sl := range availSlots {
			t, _ := time.Parse(time.RFC3339, sl.StartTime)
			msg += fmt.Sprintf("%d️⃣ %s\n", i+1, t.Format("15:04"))
		}
		return OutboundReply{Body: msg}

	case "awaiting_slot":
		intent := ParseIntent(text)
		if intent != IntentSelection {
			return OutboundReply{Body: "Please reply with a number from the list to pick a slot. 🔢"}
		}

		idx := int(text[0] - '1')
		tempSlotsRaw := session.State.Get("temp_slots")
		// Manual cast since StateData is map[string]any
		tempSlotsJson, _ := json.Marshal(tempSlotsRaw)
		var availSlots []availability.SlotDTO
		json.Unmarshal(tempSlotsJson, &availSlots)

		if idx < 0 || idx >= len(availSlots) {
			return OutboundReply{Body: "That selection is invalid. Please pick a number from the list. 🧐"}
		}

		selectedSlot := availSlots[idx]
		tStart, _ := time.Parse(time.RFC3339, selectedSlot.StartTime)
		session.State.Set("selected_start", selectedSlot.StartTime)
		session.State.Set("selected_end", selectedSlot.EndTime)
		session.CurrentStep = "awaiting_confirmation"
		s.repo.UpsertSession(ctx, session)

		return OutboundReply{Body: fmt.Sprintf("Confirming: Book appointment on %s at %s? Reply 'Yes' to confirm. ✅", tStart.Format("Jan 2"), tStart.Format("15:04"))}

	case "awaiting_confirmation":
		intent := ParseIntent(text)
		if intent != IntentConfirmation {
			return OutboundReply{Body: "Please reply with 'Yes' to confirm your booking. 👍"}
		}

		doctorIDStr := session.State.Get("doctor_id").(string)
		doctorID, _ := uuid.Parse(doctorIDStr)
		startStr := session.State.Get("selected_start").(string)
		endStr := session.State.Get("selected_end").(string)
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return OutboundReply{Body: "Something went wrong with the time format. ❌"}
		}
		end, _ := time.Parse(time.RFC3339, endStr)

		_, err = s.appointments.ScheduleAppointment(session.TenantID, *session.PatientID, doctorID, start, end, uuid.Nil) // System booked
		if err != nil {
			return OutboundReply{Body: "Something went wrong while booking. Please try again or contact the clinic. ❌"}
		}

		session.CurrentFlow = "menu"
		session.CurrentStep = "start"
		session.State.Clear()
		s.repo.UpsertSession(ctx, session)

		return OutboundReply{Body: fmt.Sprintf("Success! Your appointment is booked for %s. See you soon! 🎉", start.Format("Jan 2, 15:04"))}
	}

	return s.flowMenu(ctx, session)
}

func (s *BotService) flowCancel(ctx context.Context, session *BotSession, text string) OutboundReply {
	switch session.CurrentStep {
	case "start":
		appt, err := s.appointments.GetNextUpcomingAppointment(session.TenantID, *session.PatientID)
		if err != nil || appt == nil {
			session.CurrentFlow = "menu"
			s.repo.UpsertSession(ctx, session)
			return OutboundReply{Body: "You don't have any upcoming appointments to cancel. 🤷‍♂️"}
		}

		session.State.Set("cancel_appt_id", appt.ID.String())
		session.CurrentStep = "awaiting_confirmation"
		s.repo.UpsertSession(ctx, session)

		return OutboundReply{Body: fmt.Sprintf("You have an appointment on %s at %s with %s. Are you sure you want to cancel it? (Reply 'Yes' to confirm) ❌", appt.StartTime.Format("Jan 2"), appt.StartTime.Format("15:04"), appt.DoctorName)}

	case "awaiting_confirmation":
		intent := ParseIntent(text)
		if intent != IntentConfirmation {
			return OutboundReply{Body: "If you want to cancel, please reply with 'Yes'. Otherwise, you can say 'Menu'. ✋"}
		}

		apptIDStr := session.State.Get("cancel_appt_id").(string)
		apptID, _ := uuid.Parse(apptIDStr)

		err := s.appointments.UpdateStatus(session.TenantID, apptID, "canceled", uuid.Nil) // Log as system cancellation
		if err != nil {
			return OutboundReply{Body: "I couldn't cancel that appointment. It might be too close to the time. Please call the clinic. 🏥"}
		}

		session.CurrentFlow = "menu"
		session.CurrentStep = "start"
		session.State.Clear()
		s.repo.UpsertSession(ctx, session)

		return OutboundReply{Body: "Your appointment has been successfully canceled. 🆗"}
	}
	return s.flowMenu(ctx, session)
}

func (s *BotService) flowNext(ctx context.Context, session *BotSession, text string) OutboundReply {
	appt, err := s.appointments.GetNextUpcomingAppointment(session.TenantID, *session.PatientID)
	if err != nil || appt == nil {
		session.CurrentFlow = "menu"
		s.repo.UpsertSession(ctx, session)
		return OutboundReply{Body: "You don't have any upcoming appointments. Would you like to book one? 📅"}
	}

	session.CurrentFlow = "menu"
	s.repo.UpsertSession(ctx, session)
	return OutboundReply{Body: fmt.Sprintf("Your next appointment is on %s at %s with %s. 🗓️", appt.StartTime.Format("Mon, Jan 2"), appt.StartTime.Format("15:04"), appt.DoctorName)}
}

func (s *BotService) parseDateInput(text string) (time.Time, error) {
	lower := strings.ToLower(strings.TrimSpace(text))
	now := time.Now()

	if lower == "today" || lower == "اليوم" {
		return now, nil
	}
	if lower == "tomorrow" || lower == "بكرة" || lower == "غدا" {
		return now.AddDate(0, 0, 1), nil
	}

	// Try standard formats
	formats := []string{"2006-01-02", "02/01/2006", "02-01-2006"}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, text, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("invalid date")
}

func (s *BotService) flowReport(ctx context.Context, session *BotSession, text string) OutboundReply {
	session.CurrentFlow = "menu"
	session.CurrentStep = "start"
	s.repo.UpsertSession(ctx, session)
	return OutboundReply{Body: "Your medical reports are available securely on the Web Portal. 📄"}
}

func (s *BotService) GetPatientHistory(ctx context.Context, tenantID, patientID uuid.UUID) ([]*WhatsAppMessageDTO, error) {
	rows, err := s.repo.GetPatientMessages(ctx, tenantID, patientID)
	if err != nil {
		return nil, err
	}

	msgs := make([]*WhatsAppMessageDTO, 0, len(rows))
	for _, r := range rows {
		var providerID *string
		if r.ProviderMessageID.Valid {
			providerID = &r.ProviderMessageID.String
		}
		msgs = append(msgs, &WhatsAppMessageDTO{
			ID:                r.ID.String(),
			Direction:         r.Direction,
			PhoneNumber:       r.PhoneNumber,
			MessageType:       r.MessageType,
			Content:           r.Content,
			ProviderMessageID: providerID,
			CreatedAt:         r.CreatedAt,
		})
	}
	return msgs, nil
}

func (s *BotService) GetBotStatus(ctx context.Context, tenantID, patientID uuid.UUID) (*WhatsAppBotStatusDTO, error) {
	st, err := s.repo.GetBotStatus(ctx, tenantID, patientID)
	if err != nil {
		return nil, err
	}

	var phone *string
	if st.PhoneNumber.Valid {
		phone = &st.PhoneNumber.String
	}
	var lastInt *time.Time
	if st.LastInteraction.Valid {
		lastInt = &st.LastInteraction.Time
	}

	return &WhatsAppBotStatusDTO{
		IsReady:         st.IsReady,
		PhoneNumber:     phone,
		LastInteraction: lastInt,
		OptInStatus:     st.OptInStatus,
	}, nil
}

func (s *BotService) sendReply(ctx context.Context, session *BotSession, reply OutboundReply) {
	to := session.PhoneNumber
	if !strings.HasPrefix(to, "whatsapp:") {
		to = "whatsapp:" + to
	}

	providerMsgID, err := s.sender.Send(ctx, whatsapp.WhatsAppMessage{
		To:   to,
		Body: reply.Body,
	})

	if err != nil {
		fmt.Printf("Failed to send WhatsApp message to %s: %v\n", to, err)
		return
	}

	fmt.Printf("WhatsApp sent successfully to %s, providerMsgID=%s\n", to, providerMsgID)

	_ = s.repo.LogMessage(ctx, session.TenantID, session.PatientID, session.PhoneNumber, "outbound", "text", reply.Body, providerMsgID)
}
