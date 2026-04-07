package followup

import (
	"log"
	"time"
)

type Scheduler struct {
	svc      *Service
	interval time.Duration
	stop     chan struct{}
}

func NewScheduler(svc *Service, interval time.Duration) *Scheduler {
	return &Scheduler{
		svc:      svc,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	log.Printf("Follow-up scheduler started with interval %v", s.interval)
	ticker := time.NewTicker(s.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.runJobs()
			case <-s.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) runJobs() {
	log.Println("Running follow-up scheduled jobs...")

	// 1. Process reminders
	if err := s.svc.ProcessReminders(); err != nil {
		log.Printf("Scheduler error (ProcessReminders): %v", err)
	}

	// 2. Mark overdue as missed
	affected, err := s.svc.ProcessOverdue()
	if err != nil {
		log.Printf("Scheduler error (ProcessOverdue): %v", err)
	} else if affected > 0 {
		log.Printf("Scheduler: Marked %d follow-ups as missed", affected)
	}
}
