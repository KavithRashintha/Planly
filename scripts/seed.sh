#!/usr/bin/env bash
set -e

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080/api/v1}"
PASSWORD="Password123!"

echo "=================================================="
echo "🌱 Seeding Planly with Demo Personas"
echo "Target: $GATEWAY_URL"
echo "=================================================="

# Helper to authenticate user (login or register)
authenticate_user() {
    local email="$1"
    local full_name="$2"
    local persona="$3"
    local timezone="$4"
    local work_start="$5"
    local work_end="$6"

    echo "Authenticating $email ($persona)..." >&2

    # Try login first
    local login_resp
    login_resp=$(curl -s -X POST "$GATEWAY_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$email\",\"password\":\"$PASSWORD\"}")

    local token
    token=$(echo "$login_resp" | jq -r '.access_token // empty')

    if [ -z "$token" ]; then
        echo "User does not exist, registering $email..." >&2
        curl -s -X POST "$GATEWAY_URL/auth/register" \
            -H "Content-Type: application/json" \
            -d "{
                \"email\": \"$email\",
                \"password\": \"$PASSWORD\",
                \"full_name\": \"$full_name\",
                \"persona\": \"$persona\",
                \"timezone\": \"$timezone\",
                \"work_start\": \"$work_start\",
                \"work_end\": \"$work_end\"
            }" > /dev/null

        # Login to obtain access token
        local reg_login_resp
        reg_login_resp=$(curl -s -X POST "$GATEWAY_URL/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"email\":\"$email\",\"password\":\"$PASSWORD\"}")
        token=$(echo "$reg_login_resp" | jq -r '.access_token // empty')
    fi

    if [ -z "$token" ]; then
        echo "❌ Failed to obtain token for $email" >&2
        exit 1
    fi

    echo "$token"
}

# 1. SEED STUDENT
seed_student() {
    local token
    token=$(authenticate_user "student@planly.dev" "Sam Student" "student" "America/New_York" "08:00" "16:00")
    local auth_header="Authorization: Bearer $token"

    echo "📚 Creating Student Projects & Tasks..."
    # Project 1: High School AP Courses
    local p1_id
    p1_id=$(curl -s -X POST "$GATEWAY_URL/projects" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"name":"High School AP Courses","colour":"#3b82f6"}' | jq -r '.id')

    # Project 2: Robotics Club
    local p2_id
    p2_id=$(curl -s -X POST "$GATEWAY_URL/projects" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"name":"Robotics Club","colour":"#10b981"}' | jq -r '.id')

    # Tag: Homework
    local tag1_id
    tag1_id=$(curl -s -X POST "$GATEWAY_URL/tags" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"name":"Homework"}' | jq -r '.id')

    # Task 1: AP Calculus Problem Set
    local due_tomorrow
    due_tomorrow=$(date -u -v+1d +"%Y-%m-%dT17:00:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT17:00:00Z")
    local t1_id
    t1_id=$(curl -s -X POST "$GATEWAY_URL/tasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"project_id\": \"$p1_id\",
            \"title\": \"AP Calculus Problem Set #4\",
            \"description\": \"Complete limits and derivatives problems chapter 3\",
            \"priority\": 1,
            \"status\": \"todo\",
            \"due_at\": \"$due_tomorrow\",
            \"estimate_minutes\": 90,
            \"tag_ids\": [\"$tag1_id\"]
        }" | jq -r '.id')

    # Subtasks
    curl -s -X POST "$GATEWAY_URL/tasks/$t1_id/subtasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"title":"Solve problems 1 through 10","position":1}' > /dev/null
    curl -s -X POST "$GATEWAY_URL/tasks/$t1_id/subtasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"title":"Check solutions with study group","position":2}' > /dev/null

    # Task 2: History Reading
    local due_2d
    due_2d=$(date -u -v+2d +"%Y-%m-%dT18:00:00Z" 2>/dev/null || date -u -d "+2 days" +"%Y-%m-%dT18:00:00Z")
    curl -s -X POST "$GATEWAY_URL/tasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"project_id\": \"$p1_id\",
            \"title\": \"Read Chapter 5 of US History\",
            \"priority\": 2,
            \"status\": \"in_progress\",
            \"due_at\": \"$due_2d\",
            \"estimate_minutes\": 60
        }" > /dev/null

    # Task 3: Robotics Chassis CAD
    curl -s -X POST "$GATEWAY_URL/tasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"project_id\": \"$p2_id\",
            \"title\": \"Design intake mechanism in Onshape CAD\",
            \"priority\": 2,
            \"status\": \"todo\",
            \"estimate_minutes\": 120
        }" > /dev/null

    # Time Block
    local tb_start
    tb_start=$(date -u -v+1d +"%Y-%m-%dT14:00:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT14:00:00Z")
    local tb_end
    tb_end=$(date -u -v+1d +"%Y-%m-%dT16:00:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT16:00:00Z")
    curl -s -X POST "$GATEWAY_URL/time-blocks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Calculus Study Session\",
            \"task_id\": \"$t1_id\",
            \"starts_at\": \"$tb_start\",
            \"ends_at\": \"$tb_end\"
        }" > /dev/null

    echo "✅ Student persona seeded."
}

# 2. SEED UNDERGRADUATE
seed_undergraduate() {
    local token
    token=$(authenticate_user "undergrad@planly.dev" "Uma Undergrad" "undergraduate" "America/Los_Angeles" "09:00" "18:00")
    local auth_header="Authorization: Bearer $token"

    echo "🎓 Creating Undergraduate Projects & Tasks..."
    # Project 1: Distributed Systems
    local p1_id
    p1_id=$(curl -s -X POST "$GATEWAY_URL/projects" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"name":"CS450 Distributed Systems","colour":"#8b5cf6"}' | jq -r '.id')

    # Project 2: Research Lab
    local p2_id
    p2_id=$(curl -s -X POST "$GATEWAY_URL/projects" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"name":"Systems Research Lab","colour":"#ec4899"}' | jq -r '.id')

    # Task 1: Raft Implementation
    local due_3d
    due_3d=$(date -u -v+3d +"%Y-%m-%dT23:59:00Z" 2>/dev/null || date -u -d "+3 days" +"%Y-%m-%dT23:59:00Z")
    local t1_id
    t1_id=$(curl -s -X POST "$GATEWAY_URL/tasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"project_id\": \"$p1_id\",
            \"title\": \"Implement Raft consensus leader election in Go\",
            \"priority\": 1,
            \"status\": \"in_progress\",
            \"due_at\": \"$due_3d\",
            \"estimate_minutes\": 180
        }" | jq -r '.id')

    curl -s -X POST "$GATEWAY_URL/tasks/$t1_id/subtasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"title":"Heartbeat timer & AppendEntries RPC","position":1}' > /dev/null
    curl -s -X POST "$GATEWAY_URL/tasks/$t1_id/subtasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"title":"RequestVote RPC handler","position":2}' > /dev/null

    # Task 2: Literature Review
    curl -s -X POST "$GATEWAY_URL/tasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"project_id\": \"$p2_id\",
            \"title\": \"Write draft for ACM SIGCOMM literature review\",
            \"priority\": 2,
            \"status\": \"todo\",
            \"estimate_minutes\": 120
        }" > /dev/null

    # Time Block
    local tb_start
    tb_start=$(date -u -v+1d +"%Y-%m-%dT10:00:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT10:00:00Z")
    local tb_end
    tb_end=$(date -u -v+1d +"%Y-%m-%dT12:30:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT12:30:00Z")
    curl -s -X POST "$GATEWAY_URL/time-blocks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Deep Work: Raft RPC Implementation\",
            \"task_id\": \"$t1_id\",
            \"starts_at\": \"$tb_start\",
            \"ends_at\": \"$tb_end\"
        }" > /dev/null

    echo "✅ Undergraduate persona seeded."
}

# 3. SEED EMPLOYEE
seed_employee() {
    local token
    token=$(authenticate_user "employee@planly.dev" "Evan Employee" "employee" "UTC" "09:00" "17:00")
    local auth_header="Authorization: Bearer $token"

    echo "💼 Creating Employee Projects & Tasks..."
    # Project 1: Product Launch
    local p1_id
    p1_id=$(curl -s -X POST "$GATEWAY_URL/projects" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"name":"Q4 Product Launch","colour":"#f59e0b"}' | jq -r '.id')

    # Project 2: Platform Engineering
    local p2_id
    p2_id=$(curl -s -X POST "$GATEWAY_URL/projects" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"name":"Platform Reliability","colour":"#06b6d4"}' | jq -r '.id')

    # Task 1: Security Audit
    local due_1d
    due_1d=$(date -u -v+1d +"%Y-%m-%dT15:00:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT15:00:00Z")
    local t1_id
    t1_id=$(curl -s -X POST "$GATEWAY_URL/tasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"project_id\": \"$p2_id\",
            \"title\": \"Finalize API security audit & multi-tenant penetration tests\",
            \"priority\": 1,
            \"status\": \"todo\",
            \"due_at\": \"$due_1d\",
            \"estimate_minutes\": 150
        }" | jq -r '.id')

    curl -s -X POST "$GATEWAY_URL/tasks/$t1_id/subtasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"title":"Check cross-tenant IDOR endpoints","position":1}' > /dev/null
    curl -s -X POST "$GATEWAY_URL/tasks/$t1_id/subtasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d '{"title":"Verify token revocation on password change","position":2}' > /dev/null

    # Task 2: Release notes
    curl -s -X POST "$GATEWAY_URL/tasks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"project_id\": \"$p1_id\",
            \"title\": \"Draft release notes for v1.0.0 GA\",
            \"priority\": 2,
            \"status\": \"todo\",
            \"estimate_minutes\": 60
        }" > /dev/null

    # Time Block
    local tb_start
    tb_start=$(date -u -v+1d +"%Y-%m-%dT09:30:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT09:30:00Z")
    local tb_end
    tb_end=$(date -u -v+1d +"%Y-%m-%dT11:30:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT11:30:00Z")
    curl -s -X POST "$GATEWAY_URL/time-blocks" \
        -H "$auth_header" -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Security Audit Sprint\",
            \"task_id\": \"$t1_id\",
            \"starts_at\": \"$tb_start\",
            \"ends_at\": \"$tb_end\"
        }" > /dev/null

    echo "✅ Employee persona seeded."
}

seed_student
seed_undergraduate
seed_employee

echo "=================================================="
echo "🎉 Seed complete! All demo personas are ready:"
echo "1. Student:        student@planly.dev   (Password: $PASSWORD)"
echo "2. Undergraduate:  undergrad@planly.dev (Password: $PASSWORD)"
echo "3. Employee:       employee@planly.dev  (Password: $PASSWORD)"
echo "=================================================="
