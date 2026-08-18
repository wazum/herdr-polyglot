#!/usr/bin/env bash
# Writes the throwaway project the recording works in: one failing test for the
# prompt in the demo to ask about.
set -euo pipefail

directory="${1:-/tmp/polyglot-demo-project}"
mkdir -p "$directory"
cd "$directory"

cat > go.mod <<'GO'
module shop

go 1.26.6
GO

cat > checkout.go <<'GO'
package shop

type Item struct {
	Name  string
	Price int
	Count int
}

// Total adds up the cart in cents.
func Total(cart []Item, discount int) int {
	sum := 0
	for _, item := range cart {
		sum += item.Price * item.Count
	}
	return sum
}
GO

cat > checkout_test.go <<'GO'
package shop

import "testing"

func TestTotalTakesTheDiscountOff(t *testing.T) {
	cart := []Item{{Name: "Kaffee", Price: 250, Count: 2}}

	if total := Total(cart, 10); total != 450 {
		t.Errorf("Total is %d, want 450", total)
	}
}
GO

echo "demo project written to $directory"
