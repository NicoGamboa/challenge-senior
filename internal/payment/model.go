package payment

type Status string

const (
	StatusInitialized Status = "initialized"
	StatusPending   Status = "pending"
	StatusRejected  Status = "rejected"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type Payment struct {
	ID       string
	UserID    string
	Amount    int64
	Service   string
	Status    Status
	Reason    string
	GatewayID string
}
