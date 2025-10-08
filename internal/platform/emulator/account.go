package emulator

import (
	"errors"
	"sync"

	"github.com/shopspring/decimal"
)

type defaultAccount struct {
	balance decimal.Decimal
	mu      sync.RWMutex
}

func (a *defaultAccount) GetBalance() decimal.Decimal {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.balance
}

func (a *defaultAccount) Deposit(amount decimal.Decimal) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if amount.IsNegative() {
		return errors.New("deposit amount cannot be negative")
	}

	a.balance = a.balance.Add(amount)
	return nil
}

func (a *defaultAccount) Withdraw(amount decimal.Decimal) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if amount.IsNegative() {
		return errors.New("withdraw amount cannot be negative")
	}

	if amount.GreaterThan(a.balance) {
		return errors.New("not enough funds")
	}

	a.balance = a.balance.Sub(amount)
	return nil
}
