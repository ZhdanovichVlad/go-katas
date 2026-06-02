package domain

type Payment struct {
	ID          int64
	OrderID     string
	VendorTxID  string
	AmountCents int64
	Status      string
}
