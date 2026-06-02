package domain


type PaymentStatus string

const (
	PaymentPending PaymentStatus = "pending_vendor"
	PaymentPaid    PaymentStatus = "paid"
	PaymentFailed  PaymentStatus = "failed"
)

var SlicePaymentStatus = []string{"pending_vendor","paid","failed"}