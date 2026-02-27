#!/bin/bash

# Test script for semantic router E2E testing
# Usage: ./scripts/test-router.sh

API_URL="${API_URL:-http://localhost:8080}"

echo "🧪 Testing Semantic Router E2E"
echo "================================"
echo ""

# Health check
echo "Health Check:"
curl -s -X GET "$API_URL/test/health" | jq '.'
echo ""
echo "---"
echo ""

# Test 1: CREATE_TASK intent
echo "Test 1: CREATE_TASK - 'Làm báo cáo tuần này'"
curl -s -X POST "$API_URL/test/message" \
  -H "Content-Type: application/json" \
  -d '{"text": "Làm báo cáo tuần này"}' | jq '.'
echo ""
echo "---"
echo ""

# Test 2: SEARCH_TASK intent
echo "Test 2: SEARCH_TASK - 'Tìm task về báo cáo'"
curl -s -X POST "$API_URL/test/message" \
  -H "Content-Type: application/json" \
  -d '{"text": "Tìm task về báo cáo"}' | jq '.'
echo ""
echo "---"
echo ""

# Test 3: CONVERSATION intent
echo "Test 3: CONVERSATION - 'Hôm nay thứ mấy?'"
curl -s -X POST "$API_URL/test/message" \
  -H "Content-Type: application/json" \
  -d '{"text": "Hôm nay thứ mấy?"}' | jq '.'
echo ""
echo "---"
echo ""

# Test 4: MANAGE_CHECKLIST intent
echo "Test 4: MANAGE_CHECKLIST - 'Đánh dấu item đầu tiên là done'"
curl -s -X POST "$API_URL/test/message" \
  -H "Content-Type: application/json" \
  -d '{"text": "Đánh dấu item đầu tiên là done"}' | jq '.'
echo ""
echo "---"
echo ""

# Test 5: CREATE_TASK with context
echo "Test 5: CREATE_TASK with context - 'Thêm task mới'"
curl -s -X POST "$API_URL/test/message" \
  -H "Content-Type: application/json" \
  -d '{"text": "Thêm task mới: Review PR #123"}' | jq '.'
echo ""
echo "---"
echo ""

# Test 6: Natural language query
echo "Test 6: Natural language - 'Tôi có task nào deadline tuần này không?'"
curl -s -X POST "$API_URL/test/message" \
  -H "Content-Type: application/json" \
  -d '{"text": "Tôi có task nào deadline tuần này không?"}' | jq '.'
echo ""
echo "---"
echo ""

# Reset session
echo "Reset Session:"
curl -s -X POST "$API_URL/test/reset" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 999999999}' | jq '.'
echo ""

echo "✅ All tests completed!"
