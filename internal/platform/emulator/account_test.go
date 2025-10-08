package emulator

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeposit(t *testing.T) {
	acc := defaultAccount{balance: decimal.NewFromInt(100)}
	err := acc.Deposit(decimal.NewFromInt(200))
	require.NoError(t, err)
	assert.True(t, acc.balance.Equal(decimal.NewFromInt(300)))
}

func TestDeposit_errOnNegative(t *testing.T) {
	acc := defaultAccount{}
	err := acc.Deposit(decimal.NewFromInt(-1))
	assert.Error(t, err)
}

func TestWithdraw(t *testing.T) {
	acc := defaultAccount{balance: decimal.NewFromInt(1000)}
	err := acc.Withdraw(decimal.NewFromInt(100))
	require.NoError(t, err)
	assert.True(t, acc.balance.Equal(decimal.NewFromInt(900)))
}

func TestWithdraw_notEnoughFunds(t *testing.T) {
	acc := defaultAccount{balance: decimal.NewFromInt(1)}
	err := acc.Withdraw(decimal.NewFromInt(100))
	require.Error(t, err)
}

func TestWithdraw_errOnNegative(t *testing.T) {
	acc := defaultAccount{balance: decimal.NewFromInt(1000)}
	err := acc.Withdraw(decimal.NewFromInt(-100))
	require.Error(t, err)
}
