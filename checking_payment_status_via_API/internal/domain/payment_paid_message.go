package domain

type PaymentPaidMessage struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	Amount    int64  `json:"amount_cents"`
}
