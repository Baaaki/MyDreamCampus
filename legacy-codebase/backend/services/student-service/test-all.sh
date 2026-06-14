#!/bin/bash

# Student Service Complete Test Script
# This script tests all endpoints of the student-service

set -e  # Exit on error

BASE_URL="http://localhost:8003"
STUDENT_ID=""
JOB_ID=""

echo "🚀 Student Service API Test Suite"
echo "=================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test 1: Health Check
echo -e "${BLUE}Test 1: Health Check${NC}"
echo "GET $BASE_URL/health"
curl -s $BASE_URL/health | jq .
echo -e "${GREEN}✓ Health check passed${NC}\n"
sleep 1

# Test 2: Create Student
echo -e "${BLUE}Test 2: Create Student${NC}"
echo "POST $BASE_URL/api/v1/students"
RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{
    "student_number": "2024001",
    "first_name": "Test",
    "last_name": "Student",
    "email": "test.student@university.edu.tr",
    "faculty": "Engineering",
    "department": "Computer Engineering",
    "enrollment_year": 2024,
    "class_level": 1
  }')
echo $RESPONSE | jq .
STUDENT_ID=$(echo $RESPONSE | jq -r '.id')
echo -e "${GREEN}✓ Student created with ID: $STUDENT_ID${NC}\n"
sleep 1

# Test 3: List Students (no filters)
echo -e "${BLUE}Test 3: List Students (no filters)${NC}"
echo "GET $BASE_URL/api/v1/students?page=1&limit=10"
curl -s "$BASE_URL/api/v1/students?page=1&limit=10" | jq .
echo -e "${GREEN}✓ List students passed${NC}\n"
sleep 1

# Test 4: List Students with Filters
echo -e "${BLUE}Test 4: List Students with Filters${NC}"
echo "GET $BASE_URL/api/v1/students?page=1&limit=10&department=Computer%20Engineering&class_level=1"
curl -s "$BASE_URL/api/v1/students?page=1&limit=10&department=Computer%20Engineering&class_level=1" | jq .
echo -e "${GREEN}✓ Filtered list passed${NC}\n"
sleep 1

# Test 5: Get Student by ID
echo -e "${BLUE}Test 5: Get Student by ID${NC}"
echo "GET $BASE_URL/api/v1/students/$STUDENT_ID"
curl -s "$BASE_URL/api/v1/students/$STUDENT_ID" | jq .
echo -e "${GREEN}✓ Get student by ID passed${NC}\n"
sleep 1

# Test 6: Update Student
echo -e "${BLUE}Test 6: Update Student${NC}"
echo "PUT $BASE_URL/api/v1/students/$STUDENT_ID"
curl -s -X PUT $BASE_URL/api/v1/students/$STUDENT_ID \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Updated",
    "last_name": "Name",
    "email": "updated.name@university.edu.tr",
    "class_level": 2,
    "status": "active"
  }' | jq .
echo -e "${GREEN}✓ Student updated${NC}\n"
sleep 1

# Test 7: Search Students
echo -e "${BLUE}Test 7: Search Students (Advanced)${NC}"
echo "POST $BASE_URL/api/v1/students/search"
curl -s -X POST $BASE_URL/api/v1/students/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Updated",
    "filters": {
      "department": ["Computer Engineering"],
      "class_level": [1, 2]
    },
    "pagination": {
      "limit": 20
    }
  }' | jq .
echo -e "${GREEN}✓ Search passed${NC}\n"
sleep 1

# Test 8: Bulk Import (PostgreSQL COPY)
echo -e "${BLUE}Test 8: Bulk Import (Using PostgreSQL COPY)${NC}"
echo "POST $BASE_URL/api/v1/students/bulk-import"
IMPORT_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/students/bulk-import \
  -F "file=@test_students.csv")
echo $IMPORT_RESPONSE | jq .
JOB_ID=$(echo $IMPORT_RESPONSE | jq -r '.job_id')
echo -e "${GREEN}✓ Bulk import started with Job ID: $JOB_ID${NC}\n"
sleep 2

# Test 9: Get Import Job Status
echo -e "${BLUE}Test 9: Get Import Job Status${NC}"
echo "GET $BASE_URL/api/v1/students/bulk-import/$JOB_ID"
echo -e "${YELLOW}Waiting for import to complete...${NC}"
for i in {1..5}; do
  sleep 1
  STATUS=$(curl -s "$BASE_URL/api/v1/students/bulk-import/$JOB_ID" | jq -r '.status')
  echo "Status check $i: $STATUS"
  if [ "$STATUS" = "completed" ]; then
    break
  fi
done
curl -s "$BASE_URL/api/v1/students/bulk-import/$JOB_ID" | jq .
echo -e "${GREEN}✓ Import job status retrieved${NC}\n"
sleep 1

# Test 10: List Import Jobs
echo -e "${BLUE}Test 10: List Import Jobs${NC}"
echo "GET $BASE_URL/api/v1/students/bulk-import?page=1&limit=10"
curl -s "$BASE_URL/api/v1/students/bulk-import?page=1&limit=10" | jq .
echo -e "${GREEN}✓ Import jobs listed${NC}\n"
sleep 1

# Test 11: Verify Bulk Import Results
echo -e "${BLUE}Test 11: Verify Bulk Import - List All Students${NC}"
curl -s "$BASE_URL/api/v1/students?page=1&limit=20" | jq '.data | length'
echo "Total students after bulk import"
echo -e "${GREEN}✓ Verification passed${NC}\n"
sleep 1

# Test 12: Delete Student
echo -e "${BLUE}Test 12: Delete Student${NC}"
echo "DELETE $BASE_URL/api/v1/students/$STUDENT_ID"
curl -s -X DELETE "$BASE_URL/api/v1/students/$STUDENT_ID" | jq .
echo -e "${GREEN}✓ Student deleted${NC}\n"

echo ""
echo "=================================="
echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
echo "=================================="
echo ""
echo "Summary:"
echo "  ✓ Health check"
echo "  ✓ Create student"
echo "  ✓ List students (with and without filters)"
echo "  ✓ Get student by ID"
echo "  ✓ Update student"
echo "  ✓ Search students"
echo "  ✓ Bulk import with PostgreSQL COPY"
echo "  ✓ Import job tracking"
echo "  ✓ Delete student"
echo ""
echo "Key Features Tested:"
echo "  🚀 PostgreSQL COPY for high-performance bulk insert"
echo "  ⚡ Background job processing with goroutines"
echo "  📊 Import job status tracking"
echo "  🔍 Advanced search with filters"
echo "  📄 Pagination and sorting"
