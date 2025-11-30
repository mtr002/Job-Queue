package nats

import (
	"encoding/json"
	"fmt"

	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/nats-io/nats.go"
)

type Server struct {
	conn    *nats.Conn
	sub     *nats.Subscription
	manager *jobs.Manager
}

func NewServer(url string, manager *jobs.Manager) (*Server, error) {
	if url == "" {
		url = nats.DefaultURL
	}

	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Server{
		conn:    conn,
		manager: manager,
	}, nil
}

func (s *Server) Subscribe() error {
	sub, err := s.conn.Subscribe(JobSubmitSubject, func(msg *nats.Msg) {
		var jobMsg JobSubmissionMessage
		if err := json.Unmarshal(msg.Data, &jobMsg); err != nil {
			return
		}

		_, err := s.manager.SubmitJob(jobMsg.Type, jobMsg.Payload)
		if err != nil {
			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS: %w", err)
	}

	s.sub = sub
	return nil
}

func (s *Server) Close() {
	if s.sub != nil {
		s.sub.Unsubscribe()
	}
	if s.conn != nil {
		s.conn.Close()
	}
}

