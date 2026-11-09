package transfer

import "testing"

func TestStateMachineAllowsSettlementPath(t *testing.T) {
	path := []State{Created, Validating, FundsReserved, ReadyForSettlement, Settled, CreditPending, Completed}
	for i := 1; i < len(path); i++ {
		if err := Advance(path[i-1], path[i]); err != nil { t.Fatalf("step %s -> %s: %v", path[i-1], path[i], err) }
	}
}

func TestStateMachineRejectsCreditBeforeSettlement(t *testing.T) {
	if err := Advance(FundsReserved, Completed); err == nil { t.Fatal("expected invalid transition") }
}

func TestUnknownStateCanOnlyReconcileOrReview(t *testing.T) {
	if err := Advance(UnknownPendingReconciliation, Settled); err != nil { t.Fatal(err) }
	if err := Advance(UnknownPendingReconciliation, ManualReview); err != nil { t.Fatal(err) }
	if err := Advance(UnknownPendingReconciliation, Completed); err == nil { t.Fatal("expected completion to require receiver evidence") }
}
