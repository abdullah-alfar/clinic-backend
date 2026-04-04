package whatsappbot

import (
	"context"
	"fmt"

	"clinic-backend/internal/appointment"
	"clinic-backend/internal/whatsapp"
	"github.com/google/uuid"
)

// BotService orchestrates the conversational logic.
type BotService struct {
	repo         BotRepository
	sender       whatsapp.WhatsAppSender
	appointments *appointment.AppointmentService
	timezone     string // could be dynamic per tenant
}

func NewBotService(repo BotRepository, sender whatsapp.WhatsAppSender, appts *appointment.AppointmentService) *BotService {
	return &BotService{repo: repo, sender: sender, appointments: appts}
}

// ProcessInbound handles an incoming message from the webhook.
func (s *BotService) ProcessInbound(ctx context.Context, tenantID uuid.UUID, msg InboundMessage) error {
	phone, err := whatsapp.NormalizePhone(msg.From)
	if err != nil {
		phone = msg.From // Fallback to raw if unparseable
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

	// 3. Process Intent if we are at "start" of a flow OR a menu
	if session.CurrentFlow == "menu" || session.CurrentStep == "start" {
		intent := ParseIntent(msg.Body)
		session.CurrentFlow = intent
		session.CurrentStep = "start"
		session.State.Clear() // New flow, clear state
		s.repo.UpsertSession(ctx, session)
	}

	// 4. Handle Flow
	reply := s.handleFlow(ctx, session, msg.Body)

	// 5. Send Reply and Log
	if reply.Body != "" {
		s.sendReply(ctx, session, reply)
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
	// MVP implementation: direct them to the portal or receptionist
	session.CurrentFlow = "menu"
	session.CurrentStep = "start"
	s.repo.UpsertSession(ctx, session)
	return OutboundReply{Body: "To book a new appointment, please visit our online portal or call the reception. Automatic self-booking via WhatsApp will be available soon! 📅"}
}

func (s *BotService) flowCancel(ctx context.Context, session *BotSession, text string) OutboundReply {
	// MVP implementation: direct them to portal
	session.CurrentFlow = "menu"
	session.CurrentStep = "start"
	s.repo.UpsertSession(ctx, session)
	return OutboundReply{Body: "To cancel an appointment, please visit our online portal or call the reception. We'll be adding WhatsApp cancellations soon! ❌"}
}

func (s *BotService) flowNext(ctx context.Context, session *BotSession, text string) OutboundReply {
	session.CurrentFlow = "menu"
	session.CurrentStep = "start"
	s.repo.UpsertSession(ctx, session)
	return OutboundReply{Body: "I can't fetch your next appointment right now (Self-service coming soon!). Please check your portal. 🗓️"}
}

func (s *BotService) flowReport(ctx context.Context, session *BotSession, text string) OutboundReply {
	session.CurrentFlow = "menu"
	session.CurrentStep = "start"
	s.repo.UpsertSession(ctx, session)
	return OutboundReply{Body: "Your medical reports are available securely on the Web Portal. 📄"}
}

func (s *BotService) sendReply(ctx context.Context, session *BotSession, reply OutboundReply) {
	providerMsgID, err := s.sender.Send(ctx, whatsapp.WhatsAppMessage{
		To:   session.PhoneNumber,
		Body: reply.Body,
	})

	if err != nil {
		fmt.Printf("Failed to send WhatsApp message: %v\n", err)
		return
	}

	_ = s.repo.LogMessage(ctx, session.TenantID, session.PatientID, session.PhoneNumber, "outbound", "text", reply.Body, providerMsgID)
}
