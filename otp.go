package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	KEY     string
	Created time.Time
}

type RetentionMap map[string]OTP

func NewRetentionMap(ctx context.Context, retentionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap)

	go rm.Retention(ctx, retentionPeriod)

	return rm
}

func (rm RetentionMap) NewOTP() OTP {
	o := OTP{
		KEY:     uuid.NewString(),
		Created: time.Now(),
	}

	rm[o.KEY] = o
	return o
}

func (rm RetentionMap) VerifyOTP(otp string) bool {
	if _, ok := rm[otp]; !ok {
		return false // otp is not valid
	}

	delete(rm, otp)
	return true
}

func (rm RetentionMap) Retention(ctx context.Context, retentionPeriod time.Duration) {
	ticket := time.NewTicker(1 * time.Minute)

	defer ticket.Stop() // 리소스 해제

	for {
		select {
		case <-ticket.C:
			for _, otp := range rm {
				if otp.Created.Add(retentionPeriod).Before(time.Now()) {
					delete(rm, otp.KEY)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
