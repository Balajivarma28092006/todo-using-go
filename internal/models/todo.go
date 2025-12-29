package models

import "time"

type Todo struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (t *Todo) IsOverdue() bool {
	if t.Deadline == nil || t.Completed {
		return false
	}
	return t.Deadline.Before(time.Now()) // return bool if the time is over
}

func (t *Todo) DaysUntilDeadline() int {
	if t.Deadline == nil {
		return -1
	}

	duration := time.Until(*t.Deadline)
	return int(duration.Hours() / 24)
}

func (t *Todo) MarkComplete() {
	t.Completed = true
	now := time.Now()
	t.CompletedAt = &now
	t.UpdatedAt = now
}

func (t *Todo) MarkIncomplete() {
	t.Completed = false
	t.CompletedAt = nil
	t.UpdatedAt = time.Now()
}
