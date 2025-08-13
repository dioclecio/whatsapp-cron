package scheduler

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
	"unicode"

	"whatsapp-cron/pkg/db"
	"whatsapp-cron/pkg/events"
	"whatsapp-cron/pkg/waha"
)

type Scheduler struct {
	wahaClient *waha.Client
	eventHub   *events.Hub
	rateLimit  *RateLimit
}

type RateLimit struct {
	counter   int
	lastReset time.Time
	limit     int
}

func NewScheduler(wahaClient *waha.Client, eventHub *events.Hub, rateLimit int) *Scheduler {
	return &Scheduler{
		wahaClient: wahaClient,
		eventHub:   eventHub,
		rateLimit: &RateLimit{
			lastReset: time.Now(),
			limit:     rateLimit,
		},
	}
}

func (r *RateLimit) CanSend() bool {
	if time.Since(r.lastReset) > time.Hour {
		r.counter = 0
		r.lastReset = time.Now()
	}

	if r.counter >= r.limit {
		return false
	}

	r.counter++
	return true
}

func sanitizeInput(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) && r != '`' && r != '\'' && r != '"' {
			return r
		}
		return -1
	}, input)
}

func validateMessage(msg db.Mensagem) error {
	if msg.ID <= 0 {
		return fmt.Errorf("ID inválido")
	}
	if len(msg.Destinatario) == 0 {
		return fmt.Errorf("destinatário vazio")
	}
	if len(msg.Conteudos) == 0 {
		return fmt.Errorf("conteúdo vazio")
	}
	return nil
}

func formatChatID(destinatario string) string {
	// Remove caracteres não numéricos
	numbers := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, destinatario)

	// Se não for um número, retorna vazio
	if len(numbers) == 0 {
		return ""
	}

	// Formata como E.164 e adiciona sufixo WhatsApp
	return numbers + "@c.us"
}

func (s *Scheduler) ProcessMensagens() error {
	if !s.rateLimit.CanSend() {
		log.Println("Rate limit atingido, aguardando próximo período")
		return nil
	}

	database, err := db.LoadDB()
	if err != nil {
		s.eventHub.PublishError("DB", fmt.Sprintf("Erro ao carregar banco de dados: %v", err))
		return fmt.Errorf("erro ao carregar banco de dados: %v", err)
	}

	now := time.Now()
	currentTime := now.Format("15:04")
	currentDay := now.Weekday()

	for i, msg := range database.Mensagens {
		if msg.HorarioEnvio == currentTime {
			sendToday := false
			if len(msg.DiaSemana) == 0 {
				sendToday = true
			} else {
				for _, day := range msg.DiaSemana {
					if day == currentDay {
						sendToday = true
						break
					}
				}
			}

			if !sendToday {
				continue
			}

			if err := validateMessage(msg); err != nil {
				s.eventHub.PublishError("SCHEDULER", fmt.Sprintf("Mensagem inválida (ID=%d): %v", msg.ID, err))
				continue
			}

			chatID := formatChatID(msg.Destinatario)
			if chatID == "" {
				s.eventHub.PublishError("SCHEDULER", 
					fmt.Sprintf("Destinatário inválido (ID=%d): %s", msg.ID, msg.Destinatario))
				continue
			}

			// Escolhe conteúdo aleatório
			content := msg.Conteudos[rand.Intn(len(msg.Conteudos))]
			content = sanitizeInput(content)

			// Converte quebras de linha
			content = strings.ReplaceAll(content, "\r", "\n")

			// Envia mensagem
			if err := s.wahaClient.SendText(chatID, content); err != nil {
				s.eventHub.PublishError("SEND", 
					fmt.Sprintf("Erro ao enviar mensagem (ID=%d): %v", msg.ID, err))
				continue
			}

			// Atualiza último envio
			database.Mensagens[i].UltimoEnvio = now.Format(time.RFC3339)
			if err := db.SaveDB(database); err != nil {
				s.eventHub.PublishError("DB", 
					fmt.Sprintf("Erro ao salvar banco de dados: %v", err))
				continue
			}

			s.eventHub.PublishMessageSent(chatID, msg.ID)
		}
	}

	return nil
}
