package nats

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

const JobSubmitSubject = "jobs.submit"

type Client struct {
	conn *nats.Conn
}

func NewClient(url string) (*Client, error) {
	if url == "" {
		url = nats.DefaultURL
	}

	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Client{conn: conn}, nil
}

func (c *Client) PublishJobSubmission(msg *JobSubmissionMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal job submission message: %w", err)
	}

	if err := c.conn.Publish(JobSubmitSubject, data); err != nil {
		return fmt.Errorf("failed to publish job submission: %w", err)
	}

	return nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

