package main

import "time"

type Job struct {
	ID          string     `json:"id"`
	State       string     `json:"state"`
	InputType   string     `json:"input_type"`
	InputPath   string     `json:"input_path"`
	PodID       string     `json:"pod_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Error       string     `json:"error"`
}

type Pod struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	JobID    *string `json:"job_id"`
	JobState *string `json:"job_state"`
}

type ListJobsResponse struct {
	Jobs   []Job `json:"jobs"`
	Total  int   `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

type ListPodsResponse struct {
	Pods  []Pod `json:"pods"`
	Total int   `json:"total"`
}

type JobEvent struct {
	ID        int64     `json:"ID"`
	JobID     string    `json:"JobID"`
	Event     string    `json:"Event"`
	Timestamp time.Time `json:"Timestamp"`
	Payload   string    `json:"Payload"`

	RequestID string `json:"RequestID"`
	RunID     string `json:"RunID"`
	Producer  string `json:"Producer"`

	NATSSubject     string     `json:"NATSSubject"`
	NATSStream      string     `json:"NATSStream"`
	NATSConsumer    string     `json:"NATSConsumer"`
	NATSStreamSeq   int64      `json:"NATSStreamSeq"`
	NATSConsumerSeq int64      `json:"NATSConsumerSeq"`
	NATSDeliveries  int64      `json:"NATSDeliveries"`
	NATSStoredAt    *time.Time `json:"NATSStoredAt"`

	HandlerStatus string `json:"HandlerStatus"`
	HandlerError  string `json:"HandlerError"`
	DurationMs    int64  `json:"DurationMs"`
}

type ListJobEventsResponse struct {
	JobID  string     `json:"job_id"`
	Events []JobEvent `json:"events"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

type JobsStatus struct {
	Total     int            `json:"total"`
	ByState   map[string]int `json:"by_state"`
	Active    int            `json:"active"`
	Completed int            `json:"completed"`
	Failed    int            `json:"failed"`
}

type PodsStatus struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
}

type SystemStatusResponse struct {
	Jobs JobsStatus `json:"jobs"`
	Pods PodsStatus `json:"pods"`
}
