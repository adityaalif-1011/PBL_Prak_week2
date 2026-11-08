#!/bin/bash

# ============================================================
# TEST RBAC - API STUDENTS (FIXED)
# ============================================================

B="http://localhost:3000/api/v1"
J="Content-Type: application/json"

echo "============================================================"
echo "  LOGIN 3 AKUN"
echo "============================================================"

ADMIN_RESP=$(curl -s -X POST $B/auth/login -H "$J" \
  -d '{"username":"admin1","password":"rahasia123"}')
TOKEN_ADMIN=$(echo "$ADMIN_RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
echo "--- ADMIN LOGIN ---"
echo "$ADMIN_RESP"

STAFF_RESP=$(curl -s -X POST $B/auth/login -H "$J" \
  -d '{"username":"staff1","password":"rahasia123"}')
TOKEN_STAFF=$(echo "$STAFF_RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
echo "--- STAFF LOGIN ---"
echo "$STAFF_RESP"

USER_RESP=$(curl -s -X POST $B/auth/login -H "$J" \
  -d '{"username":"user1","password":"rahasia123"}')
TOKEN_USER=$(echo "$USER_RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
echo "--- USER LOGIN ---"
echo "$USER_RESP"

echo ""
echo "============================================================"
echo "  PERSIAPAN: Bikin 2 Student + assign owner"
echo "============================================================"

# Hapus semua students dulu
psql -U adityaalifsantoso -d praktikum_backend -c "DELETE FROM students;" > /dev/null
psql -U adityaalifsantoso -d praktikum_backend -c "ALTER SEQUENCE students_id_seq RESTART WITH 1;" > /dev/null

# Student A - milik staff
echo ""
echo "--- STAFF POST Student A (id=1) ---"
curl -s -X POST $B/students \
  -H "Authorization: Bearer $TOKEN_STAFF" \
  -H "$J" \
  -d '{"nim":"111111","name":"Student A","grade":80,"is_active":true}'
echo ""

# Student B - milik user1 (bikin pakai staff, lalu update owner via DB)
echo ""
echo "--- STAFF POST Student B (id=2) ---"
curl -s -X POST $B/students \
  -H "Authorization: Bearer $TOKEN_STAFF" \
  -H "$J" \
  -d '{"nim":"222222","name":"Student B","grade":85,"is_active":true}'
echo ""

# Update owner student id=2 jadi user1 (id=12)
echo ""
echo "--- UPDATE owner student id=2 jadi user1 (id=12) ---"
psql -U adityaalifsantoso -d praktikum_backend -c "UPDATE students SET owner_id=12 WHERE id=2;"

# Verifikasi
echo ""
echo "--- CEK DATA STUDENTS ---"
psql -U adityaalifsantoso -d praktikum_backend -c "SELECT id, nim, name, owner_id FROM students ORDER BY id;"

echo ""
echo "============================================================"
echo "  MATRIKS HAK AKSES"
echo "============================================================"

echo ""
echo "########## 1. GET /students ##########"
echo "--- ADMIN ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" $B/students -H "Authorization: Bearer $TOKEN_ADMIN"
echo "--- STAFF ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" $B/students -H "Authorization: Bearer $TOKEN_STAFF"
echo "--- USER (non-owner) ---"
curl -s -w "\nStatus: %{http_code}\n" $B/students -H "Authorization: Bearer $TOKEN_USER"

echo ""
echo "########## 2. POST /students ##########"
echo "--- ADMIN ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" -X POST $B/students -H "Authorization: Bearer $TOKEN_ADMIN" -H "$J" -d '{"nim":"333333","name":"Admin Create","grade":70,"is_active":true}'
echo "--- STAFF ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" -X POST $B/students -H "Authorization: Bearer $TOKEN_STAFF" -H "$J" -d '{"nim":"444444","name":"Staff Create","grade":70,"is_active":true}'
echo "--- USER (non-owner) ---"
curl -s -w "\nStatus: %{http_code}\n" -X POST $B/students -H "Authorization: Bearer $TOKEN_USER" -H "$J" -d '{"nim":"555555","name":"User Create","grade":70,"is_active":true}'

echo ""
echo "########## 3. GET /students/1 (MILIK STAFF) ##########"
echo "--- ADMIN (non-owner) ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" $B/students/1 -H "Authorization: Bearer $TOKEN_ADMIN"
echo "--- STAFF (owner) ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" $B/students/1 -H "Authorization: Bearer $TOKEN_STAFF"
echo "--- USER (non-owner) ---"
curl -s -w "\nStatus: %{http_code}\n" $B/students/1 -H "Authorization: Bearer $TOKEN_USER"

echo ""
echo "########## 4. GET /students/2 (MILIK USER1) ##########"
echo "--- ADMIN (non-owner) ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" $B/students/2 -H "Authorization: Bearer $TOKEN_ADMIN"
echo "--- STAFF (non-owner) ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" $B/students/2 -H "Authorization: Bearer $TOKEN_STAFF"
echo "--- USER (owner) ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" $B/students/2 -H "Authorization: Bearer $TOKEN_USER"

echo ""
echo "########## 5. PUT /students/1 (MILIK STAFF) ##########"
echo "--- ADMIN (non-owner, punya update:any) ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" -X PUT $B/students/1 -H "Authorization: Bearer $TOKEN_ADMIN" -H "$J" -d '{"nim":"111111","name":"Updated by Admin","grade":90,"is_active":true}'
echo "--- STAFF (owner) ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" -X PUT $B/students/1 -H "Authorization: Bearer $TOKEN_STAFF" -H "$J" -d '{"nim":"111111","name":"Updated by Staff","grade":90,"is_active":true}'
echo "--- USER (non-owner) ---"
curl -s -w "\nStatus: %{http_code}\n" -X PUT $B/students/1 -H "Authorization: Bearer $TOKEN_USER" -H "$J" -d '{"nim":"111111","name":"Hacked","grade":100,"is_active":true}'

echo ""
echo "########## 6. DELETE /students/1 ##########"
echo "--- STAFF (gak punya delete) ---"
curl -s -w "\nStatus: %{http_code}\n" -X DELETE $B/students/1 -H "Authorization: Bearer $TOKEN_STAFF"
echo "--- USER (gak punya delete) ---"
curl -s -w "\nStatus: %{http_code}\n" -X DELETE $B/students/1 -H "Authorization: Bearer $TOKEN_USER"
echo "--- ADMIN (punya delete) ---"
curl -s -o /dev/null -w "Status: %{http_code}\n" -X DELETE $B/students/1 -H "Authorization: Bearer $TOKEN_ADMIN"

echo ""
echo "============================================================"
echo "  PENGUJIAN NEGATIF"
echo "============================================================"

echo ""
echo "########## NEGATIF 1: Mass Assignment owner_id ##########"
echo "--- Staff POST dengan owner_id:999 ---"
curl -s -w "\nStatus: %{http_code}\n" -X POST $B/students -H "Authorization: Bearer $TOKEN_STAFF" -H "$J" -d '{"nim":"666666","name":"Mass Assign Test","grade":75,"is_active":true,"owner_id":999}'

echo ""
echo "--- Cek DB: owner_id harus ID staff (11), bukan 999 ---"
psql -U adityaalifsantoso -d praktikum_backend -c "SELECT id, nim, owner_id FROM students WHERE nim='666666';"

echo ""
echo "########## NEGATIF 2: PUT data milik orang lain ##########"
echo "--- User PUT student id=1 (milik staff) ---"
curl -s -w "\nStatus: %{http_code}\n" -X PUT $B/students/1 -H "Authorization: Bearer $TOKEN_USER" -H "$J" -d '{"nim":"111111","name":"Hacked","grade":100,"is_active":true}'

echo ""
echo "============================================================"
echo "  SELESAI"
echo "============================================================"