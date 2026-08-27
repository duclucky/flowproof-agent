package model

import "time"

type RunStatus string

const (
	RunQueued            RunStatus = "queued"
	RunRunning           RunStatus = "running"
	RunFailedRecoverable RunStatus = "failed_recoverable"
	RunSucceeded         RunStatus = "succeeded"
	RunFailed            RunStatus = "failed"
	RunCancelled         RunStatus = "cancelled"
)

type ActionKind string

const (
	ActionNavigate ActionKind = "navigate"
	ActionClick    ActionKind = "click"
	ActionFill     ActionKind = "fill"
	ActionPress    ActionKind = "press"
	ActionWait     ActionKind = "wait"
	ActionAssert   ActionKind = "assert"
	ActionComplete ActionKind = "complete"
)

type CreateTestRequest struct {
	TargetURL string `json:"targetUrl"`
	Objective string `json:"objective"`
}

type TestDefinition struct {
	ID        string    `json:"id"`
	TargetURL string    `json:"targetUrl"`
	Objective string    `json:"objective"`
	CreatedAt time.Time `json:"createdAt"`
}

type BrowserObservation struct {
	CurrentURL        string `json:"currentUrl"`
	VisibleText       string `json:"visibleText"`
	Screenshot        []byte `json:"-"`
	DOM               string `json:"-"`
	AttemptedSelector string `json:"attemptedSelector,omitempty"`
	FallbackSelector  string `json:"fallbackSelector,omitempty"`
}

type RecoverableStepError struct {
	Step             string `json:"step"`
	Selector         string `json:"selector"`
	FallbackSelector string `json:"fallbackSelector"`
	Message          string `json:"message"`
}

func (e *RecoverableStepError) Error() string {
	if e == nil {
		return "recoverable browser step failed"
	}
	return e.Message
}

type FailureAnalysis struct {
	Step             string `json:"step"`
	FailedSelector   string `json:"failedSelector"`
	FallbackSelector string `json:"fallbackSelector"`
	Explanation      string `json:"explanation"`
	Recoverable      bool   `json:"recoverable"`
}

type CreateRunRequest struct {
	TargetURL string `json:"targetUrl"`
	Objective string `json:"objective"`
	MaxSteps  int    `json:"maxSteps,omitempty"`
}

type Run struct {
	ID                string           `json:"id"`
	TestID            string           `json:"testId,omitempty"`
	TargetURL         string           `json:"targetUrl"`
	Objective         string           `json:"objective"`
	Status            RunStatus        `json:"status"`
	CurrentURL        string           `json:"currentUrl,omitempty"`
	StepCount         int              `json:"stepCount"`
	MaxSteps          int              `json:"maxSteps"`
	Summary           string           `json:"summary,omitempty"`
	Failure           string           `json:"failure,omitempty"`
	FailureAnalysis   *FailureAnalysis `json:"failureAnalysis,omitempty"`
	RecoveredSelector string           `json:"recoveredSelector,omitempty"`
	Events            []RunEvent       `json:"events"`
	Evidence          []Evidence       `json:"evidence,omitempty"`
	CreatedAt         time.Time        `json:"createdAt"`
	UpdatedAt         time.Time        `json:"updatedAt"`
	CompletedAt       *time.Time       `json:"completedAt,omitempty"`
}

type RunEvent struct {
	Seq         int       `json:"seq"`
	At          time.Time `json:"at"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Action      *Action   `json:"action,omitempty"`
	Observation *Snapshot `json:"observation,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type Evidence struct {
	Step       int       `json:"step"`
	Kind       string    `json:"kind"`
	Label      string    `json:"label"`
	DataURL    string    `json:"dataUrl,omitempty"`
	Text       string    `json:"text,omitempty"`
	CapturedAt time.Time `json:"capturedAt"`
}

type Snapshot struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	DOM         string `json:"dom"`
	VisibleText string `json:"visibleText"`
	Screenshot  []byte `json:"-"`
}

type Action struct {
	Kind       ActionKind `json:"kind"`
	Selector   string     `json:"selector,omitempty"`
	Value      string     `json:"value,omitempty"`
	URL        string     `json:"url,omitempty"`
	Key        string     `json:"key,omitempty"`
	Text       string     `json:"text,omitempty"`
	Reason     string     `json:"reason"`
	Evidence   string     `json:"evidence,omitempty"`
	WaitMS     int        `json:"waitMs,omitempty"`
	Confidence float64    `json:"confidence,omitempty"`
}

type PlannerInput struct {
	RunID     string     `json:"runId"`
	TargetURL string     `json:"targetUrl"`
	Objective string     `json:"objective"`
	Step      int        `json:"step"`
	MaxSteps  int        `json:"maxSteps"`
	Snapshot  Snapshot   `json:"snapshot"`
	Events    []RunEvent `json:"events"`
	LastError string     `json:"lastError,omitempty"`
}
