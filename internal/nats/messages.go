package nats

type JobSubmissionMessage struct {
	Type        string `json:"type"`
	Payload     string `json:"payload"`
	MaxAttempts int    `json:"max_attempts"`
}

type JobStatusMessage struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Result  string `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

