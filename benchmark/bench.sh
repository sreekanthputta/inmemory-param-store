#!/bin/bash
# Benchmark using autocannon
# Install: npm install -g autocannon

URL="${1:-http://localhost:8847}"
DURATION="${2:-10}"
CONNECTIONS="${3:-100}"

echo "Parameter Store Benchmark (autocannon)"
echo "======================================="
echo "URL:         $URL"
echo "Duration:    ${DURATION}s"
echo "Connections: $CONNECTIONS"
echo ""

# Seed test data
echo "Seeding test key..."
curl -s -X POST "$URL/api/update" \
  -H "Content-Type: application/json" \
  -d '{"updates":[{"key":"__bench_read_test","value":"test_value","type":"text"}]}' > /dev/null

echo ""
echo "=== WRITE (POST /api/update) ==="
npx autocannon -c $CONNECTIONS -d $DURATION -m POST \
  -H "Content-Type: application/json" \
  -b '{"updates":[{"key":"__bench_write_test","value":"bench_value","type":"text"}]}' \
  "$URL/api/update"

echo ""
echo "=== GET (GET /api/get) ==="
npx autocannon -c $CONNECTIONS -d $DURATION \
  "$URL/api/get?key=__bench_read_test"

echo ""
echo "=== LIST (GET /api/list) ==="
npx autocannon -c $CONNECTIONS -d $DURATION \
  "$URL/api/list"

echo ""
echo "Cleaning up test data..."
curl -s -X POST "$URL/api/update" \
  -H "Content-Type: application/json" \
  -d '{"updates":[
    {"key":"__bench_read_test","value":"","type":"text","is_delete":true},
    {"key":"__bench_write_test","value":"","type":"text","is_delete":true}
  ]}' > /dev/null
echo "Done."
