#!/usr/bin/env bash
set -euo pipefail

# ─── Configuration ───────────────────────────────────────────────────────────
LEDGER_URL="${LEDGER_URL:-http://localhost:8080}"
PROCESSOR_URL="${PROCESSOR_URL:-http://localhost:8082}"

GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

PASS=0
FAIL=0

# ─── Helpers ─────────────────────────────────────────────────────────────────
log()  { echo -e "${CYAN}▶ $1${NC}"; }
pass() { echo -e "  ${GREEN}✔ $1${NC}"; PASS=$((PASS + 1)); }
fail() { echo -e "  ${RED}✘ $1${NC}"; FAIL=$((FAIL + 1)); }

assert_eq() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    pass "$label  (got $actual)"
  else
    fail "$label  (expected $expected, got $actual)"
  fi
}

json_field() {
  # Extract a JSON field value using pure bash (no jq dependency).
  # Works for simple flat JSON: {"key":"value",...}
  local json="$1" field="$2"
  echo "$json" | python3 -c "import sys,json; print(json.loads(sys.stdin.read()).get('$field',''))" 2>/dev/null
}

# ─── Smoke Test ──────────────────────────────────────────────────────────────
echo -e "\n${BOLD}🏦  Core Banking Engine — Smoke Test${NC}\n"

# 1. Health checks
log "Health checks"
for svc in "$LEDGER_URL" "${CORE_URL:-http://localhost:8081}" "$PROCESSOR_URL"; do
  status=$(curl -s -o /dev/null -w "%{http_code}" "$svc/health" 2>/dev/null || echo "000")
  if [ "$status" = "200" ]; then
    pass "$svc/health → 200"
  else
    fail "$svc/health → $status"
  fi
done

# 2. Create accounts
log "Creating accounts"
ALICE_JSON=$(curl -s -X POST "$LEDGER_URL/accounts" \
  -H "Content-Type: application/json" \
  -d '{"owner":"Alice","currency":"EUR"}')
ALICE_ID=$(json_field "$ALICE_JSON" "id")
ALICE_BAL=$(json_field "$ALICE_JSON" "balance")
assert_eq "Alice created with balance 0" "0" "${ALICE_BAL%.*}"  # strip decimal

BOB_JSON=$(curl -s -X POST "$LEDGER_URL/accounts" \
  -H "Content-Type: application/json" \
  -d '{"owner":"Bob","currency":"EUR"}')
BOB_ID=$(json_field "$BOB_JSON" "id")
BOB_BAL=$(json_field "$BOB_JSON" "balance")
assert_eq "Bob created with balance 0" "0" "${BOB_BAL%.*}"

echo -e "  Alice ID: $ALICE_ID"
echo -e "  Bob   ID: $BOB_ID"

# 3. Deposit €500 to Alice via Processor
log "Deposit €500 to Alice (via Processor)"
DEP_JSON=$(curl -s -X POST "$PROCESSOR_URL/process/deposit" \
  -H "Content-Type: application/json" \
  -d "{\"account_id\":\"$ALICE_ID\",\"amount\":500.00}")
DEP_STATUS=$(json_field "$DEP_JSON" "status")
assert_eq "Deposit status" "COMPLETED" "$DEP_STATUS"

# 4. Verify Alice balance = 500
log "Verify Alice balance after deposit"
ALICE_JSON=$(curl -s "$LEDGER_URL/accounts/$ALICE_ID")
ALICE_BAL=$(json_field "$ALICE_JSON" "balance")
assert_eq "Alice balance" "500" "${ALICE_BAL%.*}"

# 5. Transfer €150 from Alice to Bob via Processor
log "Transfer €150 Alice → Bob (via Processor)"
TRF_JSON=$(curl -s -X POST "$PROCESSOR_URL/process/transfer" \
  -H "Content-Type: application/json" \
  -d "{\"from_account_id\":\"$ALICE_ID\",\"to_account_id\":\"$BOB_ID\",\"amount\":150.00}")
TRF_STATUS=$(json_field "$TRF_JSON" "status")
assert_eq "Transfer status" "COMPLETED" "$TRF_STATUS"

# 6. Verify final balances
log "Verify final balances"
ALICE_JSON=$(curl -s "$LEDGER_URL/accounts/$ALICE_ID")
ALICE_BAL=$(json_field "$ALICE_JSON" "balance")
assert_eq "Alice final balance" "350" "${ALICE_BAL%.*}"

BOB_JSON=$(curl -s "$LEDGER_URL/accounts/$BOB_ID")
BOB_BAL=$(json_field "$BOB_JSON" "balance")
assert_eq "Bob final balance" "150" "${BOB_BAL%.*}"

# 7. Negative test: overdraft
log "Negative test: overdraft (transfer €9999 from Bob)"
OD_JSON=$(curl -s -X POST "$PROCESSOR_URL/process/transfer" \
  -H "Content-Type: application/json" \
  -d "{\"from_account_id\":\"$BOB_ID\",\"to_account_id\":\"$ALICE_ID\",\"amount\":9999.00}")
OD_STATUS=$(json_field "$OD_JSON" "status")
if [ "$OD_STATUS" = "COMPLETED" ]; then
  fail "Overdraft should have been rejected"
else
  pass "Overdraft correctly rejected  (status=$OD_STATUS)"
fi

# 8. Negative test: self-transfer
log "Negative test: self-transfer"
SELF_JSON=$(curl -s -X POST "$PROCESSOR_URL/process/transfer" \
  -H "Content-Type: application/json" \
  -d "{\"from_account_id\":\"$ALICE_ID\",\"to_account_id\":\"$ALICE_ID\",\"amount\":10.00}")
SELF_ERR=$(json_field "$SELF_JSON" "error")
if [ -n "$SELF_ERR" ]; then
  pass "Self-transfer correctly rejected"
else
  fail "Self-transfer should have been rejected"
fi

# 9. Ledger entries
log "Verify ledger entries for Alice"
ENTRIES=$(curl -s "$LEDGER_URL/ledger/entries/$ALICE_ID")
ENTRY_COUNT=$(echo "$ENTRIES" | python3 -c "import sys,json; print(len(json.loads(sys.stdin.read())))" 2>/dev/null)
assert_eq "Alice has 2 ledger entries (deposit + transfer debit)" "2" "$ENTRY_COUNT"

# ─── Summary ─────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  Results:  ${GREEN}$PASS passed${NC}  ${RED}$FAIL failed${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
