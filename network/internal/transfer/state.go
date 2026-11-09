package transfer

import "fmt"

type State string

const (
	Created State = "CREATED"
	Validating State = "VALIDATING"
	FundsReserved State = "FUNDS_RESERVED"
	ReadyForSettlement State = "READY_FOR_SETTLEMENT"
	Settled State = "SETTLED"
	CreditPending State = "CREDIT_PENDING"
	Completed State = "COMPLETED"
	Rejected State = "REJECTED"
	Expired State = "EXPIRED"
	UnknownPendingReconciliation State = "UNKNOWN_PENDING_RECONCILIATION"
	ManualReview State = "MANUAL_REVIEW"
	Refunded State = "REFUNDED"
)

var transitions = map[State]map[State]bool{
	Created: {Validating: true},
	Validating: {FundsReserved: true, Rejected: true, Expired: true},
	FundsReserved: {ReadyForSettlement: true, Rejected: true, Expired: true},
	ReadyForSettlement: {Settled: true, UnknownPendingReconciliation: true},
	Settled: {CreditPending: true, UnknownPendingReconciliation: true, Refunded: true},
	CreditPending: {CreditPending: true, Completed: true},
	UnknownPendingReconciliation: {Settled: true, ManualReview: true},
}

func Advance(from, to State) error {
	if transitions[from][to] { return nil }
	return fmt.Errorf("invalid transfer transition: %s -> %s", from, to)
}
