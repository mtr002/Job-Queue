package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/mtr002/Job-Queue/internal/interfaces"
	"github.com/mtr002/Job-Queue/proto"
)

type Client struct {
	conn   *grpc.ClientConn
	client proto.WorkerServiceClient
}

func NewClient(addr string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: proto.NewWorkerServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) SubmitJob(jobType, payload string, maxAttempts int) (*interfaces.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SubmitJob(ctx, &proto.SubmitJobRequest{
		Type:        jobType,
		Payload:     payload,
		MaxAttempts: int32(maxAttempts),
	})
	if err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		log.Printf("Failed to parse created_at: %v", err)
		createdAt = time.Now()
	}

	job := &interfaces.Job{
		ID:          resp.JobId,
		Type:        jobType,
		Payload:     payload,
		Status:      interfaces.JobStatus(resp.Status),
		MaxAttempts: maxAttempts,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}

	return job, nil
}

func (c *Client) GetJobStatus(jobID string) (*interfaces.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetJobStatus(ctx, &proto.GetJobRequest{
		JobId: jobID,
	})
	if err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	updatedAt, err := time.Parse(time.RFC3339, resp.UpdatedAt)
	if err != nil {
		updatedAt = time.Now()
	}

	job := &interfaces.Job{
		ID:          resp.JobId,
		Type:        resp.Type,
		Payload:     resp.Payload,
		Status:      interfaces.JobStatus(resp.Status),
		Result:      resp.Result,
		Error:       resp.Error,
		Attempts:    int(resp.Attempts),
		MaxAttempts: int(resp.MaxAttempts),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	return job, nil
}
