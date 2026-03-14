#!/bin/bash
# Demo: parameter store operations
# Run after: go run .

BASE_URL="http://localhost:8847"

echo "=== 1. Insert parameters ==="
curl -s -X POST "${BASE_URL}/api/update" \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"key": "db_host", "value": "localhost", "type": "text"},
      {"key": "db_port", "value": "5432", "type": "text"},
      {"key": "db_password", "value": "secret123", "type": "password"}
    ]
  }' | jq .

echo ""
echo "=== 2. List (passwords masked) ==="
curl -s "${BASE_URL}/api/list" | jq .

echo ""
echo "=== 3. Update db_host ==="
curl -s -X POST "${BASE_URL}/api/update" \
  -H "Content-Type: application/json" \
  -d '{"updates": [{"key": "db_host", "value": "prod.example.com", "type": "text"}]}' | jq .

echo ""
echo "=== 4. Delete db_port ==="
curl -s -X POST "${BASE_URL}/api/update" \
  -H "Content-Type: application/json" \
  -d '{"updates": [{"key": "db_port", "is_delete": true}]}' | jq .

echo ""
echo "=== 5. List after changes ==="
curl -s "${BASE_URL}/api/list" | jq .

echo ""
echo "=== 6. Data file ==="
cat data.jsonl

echo ""
echo "Web UI: http://localhost:8847"
