package usecase

import (
	"context"
	"sync"

	"github.com/ZhdanovichVlad/go-katas/checking_payment_status_via_API/internal/domain"
)

const (
	semaphoreSize = 5
)

type storage interface {
	GetPendingPaymentsAndLockRows(ctx context.Context) ([]string, error)
	UpdatePaymentStatusAndUnlockRows(ctx context.Context, vendorTxID []string, status domain.PaymentStatus) error
	GetStackRows(ctx context.Context) ([]string, error)
}


type fulfillmentQueue interface {
	EnqueuePaymentPaid(ctx context.Context, msg domain.PaymentPaidMessage) error
}

type vendorCheck interface {
	GetTxStatus(ctx context.Context, txId string) (string, error)
}



type usecase struct {
	storage storage
	vendor  vendorCheck
	brocker fulfillmentQueue
}

func New(storage storage) *usecase {
	return &usecase{storage:storage}
}



func (us *usecase) CheckPaymentStatus(ctx context.Context) error {
	ids, err := us.storage.GetPendingPaymentsAndLockRows(ctx)
	if err != nil {
		return err
	}

	answers := make(map[string][]string, 4)
	answersMutes := sync.Mutex{}
	for _, status := range(domain.SlicePaymentStatus){
		answers[status]=make([]string, 0)
	}

	semaphore := make(chan struct{}, semaphoreSize) 
	wg := sync.WaitGroup{}

	for _, value := range(ids){
		semaphore <-struct{}{}
		wg.Go(func() {
			status, err := us.vendor.GetTxStatus(ctx, value)
			if err != nil {
				// todo : logs
				answersMutes.Lock()
				answers[string(domain.PaymentFailed)] = append(answers[string(domain.PaymentFailed)], value)
				answersMutes.Unlock()
			}

			switch status {
			case string(domain.PaymentPaid):
				answersMutes.Lock()
				answers[string(domain.PaymentPaid)] = append(answers[string(domain.PaymentPaid)], value)
				answersMutes.Unlock()
			case string(domain.PaymentPending):	
				answersMutes.Lock()
				answers[string(domain.PaymentPending)] = append(answers[string(domain.PaymentPending)], value)
				answersMutes.Unlock()
			default:
				answersMutes.Lock()
				answers[string(domain.PaymentFailed)] = append(answers[string(domain.PaymentFailed)], value)
				answersMutes.Unlock()
			}
			
			<-semaphore
		})
	}
	wg.Wait()

	for status, ids := range answers {
		err = us.storage.UpdatePaymentStatusAndUnlockRows(ctx, ids, domain.PaymentStatus(status))
		if err!= nil {
			return err
		}
	}

	return nil
}


func (us *usecase) CheckStack(ctx context.Context) error {
	ids, err := us.storage.GetStackRows(ctx)
	if err != nil {
		return err
	}

	answers := make(map[string][]string, 4)
	answersMutes := sync.Mutex{}
	for _, status := range(domain.SlicePaymentStatus){
		answers[status]=make([]string, 0)
	}

	semaphore := make(chan struct{}, semaphoreSize) 
	wg := sync.WaitGroup{}

	for _, value := range(ids){
		semaphore <-struct{}{}
		wg.Go(func() {
			status, err := us.vendor.GetTxStatus(ctx, value)
			if err != nil {
				answersMutes.Lock()
				answers[string(domain.PaymentFailed)] = append(answers[string(domain.PaymentFailed)], value)
				answersMutes.Unlock()
			}

			switch status {
			case string(domain.PaymentPaid):
				answersMutes.Lock()
				answers[string(domain.PaymentPaid)] = append(answers[string(domain.PaymentPaid)], value)
				answersMutes.Unlock()
			case string(domain.PaymentPending):	
				answersMutes.Lock()
				answers[string(domain.PaymentPending)] = append(answers[string(domain.PaymentPending)], value)
				answersMutes.Unlock()
			default:
				answersMutes.Lock()
				answers[string(domain.PaymentFailed)] = append(answers[string(domain.PaymentFailed)], value)
				answersMutes.Unlock()
			}
			
			<-semaphore
		})
	}

	for status, ids := range answers {
		err = us.storage.UpdatePaymentStatusAndUnlockRows(ctx, ids, domain.PaymentStatus(status))
		if err!= nil {
			return err
		}
	}

	return nil
}