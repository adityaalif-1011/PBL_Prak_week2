#!/bin/bash

B="http://localhost:3000/api/v1"
J="Content-Type: application/json"

echo "=========================================="
echo "  TEST AUTHENTICATION - SEMUA ENDPOINT"
echo "=========================================="
echo ""

# ==========================================
# 1. REGISTER
# ==========================================
echo "=== 1. REGISTER USER BARU ==="
curl -s -i -X POST $B/auth/register -H "$J" \
  -d '{"username":"sari","email":"sari@example.com","password":"rahasia123"}' | head -20
echo ""
echo ""

# ==========================================
# 2. REGISTER PASSWORD LEMAH
# ==========================================
echo "=== 2. REGISTER PASSWORD LEMAH (harus 422) ==="
curl -s -i -X POST $B/auth/register -H "$J" \
  -d '{"username":"budi","email":"budi@example.com","password":"password1"}' | head -20
echo ""
echo ""

# ==========================================
# 3. LOGIN SALAH
# ==========================================
echo "=== 3. LOGIN PASSWORD SALAH (harus 401) ==="
curl -s -i -X POST $B/auth/login -H "$J" \
  -d '{"username":"sari","password":"salahsekali9"}' | head -20
echo ""
echo ""

# ==========================================
# 4. LOGIN USERNAME TIDAK ADA
# ==========================================
echo "=== 4. LOGIN USERNAME TIDAK ADA (harus 401, pesan sama) ==="
curl -s -i -X POST $B/auth/login -H "$J" \
  -d '{"username":"tidakada","password":"salahsekali9"}' | head -20
echo ""
echo ""

# ==========================================
# 5. LOGIN BENAR
# ==========================================
echo "=== 5. LOGIN BENAR (harus 200) ==="
LOGIN_RESPONSE=$(curl -s -X POST $B/auth/login -H "$J" \
  -d '{"username":"sari","password":"rahasia123"}')
echo "$LOGIN_RESPONSE"
echo ""

# Extract token
TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH=$(echo "$LOGIN_RESPONSE" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
echo ""
echo ""

# ==========================================
# 6. AKSES TANPA TOKEN
# ==========================================
echo "=== 6. AKSES /students TANPA TOKEN (harus 401) ==="
curl -s -i $B/students | head -10
echo ""
echo ""

# ==========================================
# 7. AKSES DENGAN TOKEN
# ==========================================
echo "=== 7. AKSES /students DENGAN TOKEN (harus 200) ==="
curl -s -i $B/students -H "Authorization: Bearer $TOKEN" | head -10
echo ""
echo ""

# ==========================================
# 8. AKSES /auth/me
# ==========================================
echo "=== 8. AKSES /auth/me DENGAN TOKEN (harus 200) ==="
curl -s -i $B/auth/me -H "Authorization: Bearer $TOKEN" | head -20
echo ""
echo ""

# ==========================================
# 9. TOKEN DIUBAH 1 KARAKTER
# ==========================================
echo "=== 9. TOKEN DIUBAH 1 KARAKTER (harus 401) ==="
curl -s -i $B/students -H "Authorization: Bearer ${TOKEN}X" | head -10
echo ""
echo ""

# ==========================================
# 10. REFRESH TOKEN
# ==========================================
echo "=== 10. REFRESH TOKEN (harus 200) ==="
REFRESH_RESPONSE=$(curl -s -X POST $B/auth/refresh -H "$J" \
  -d "{\"refresh_token\":\"$REFRESH\"}")
echo "$REFRESH_RESPONSE"
NEW_REFRESH=$(echo "$REFRESH_RESPONSE" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
echo ""
echo ""

# ==========================================
# 11. REFRESH PAKAI TOKEN LAMA (HARUS GAGAL)
# ==========================================
echo "=== 11. REFRESH PAKAI TOKEN LAMA (harus 401) ==="
curl -s -i -X POST $B/auth/refresh -H "$J" \
  -d "{\"refresh_token\":\"$REFRESH\"}" | head -10
echo ""
echo ""

# ==========================================
# 12. LOGOUT
# ==========================================
echo "=== 12. LOGOUT (harus 200) ==="
curl -s -i -X POST $B/auth/logout -H "$J" \
  -d "{\"refresh_token\":\"$NEW_REFRESH\"}" | head -10
echo ""
echo ""

# ==========================================
# 13. BRUTE FORCE (6 KALI)
# ==========================================
echo "=== 13. BRUTE FORCE 6 KALI (harus 429 di ke-6) ==="
for i in 1 2 3 4 5 6; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $B/auth/login -H "$J" \
    -d '{"username":"sari","password":"salahsekali9"}')
  echo "Percobaan ke-$i: $STATUS"
done
echo ""
echo ""

# ==========================================
# 14. MASS ASSIGNMENT
# ==========================================
echo "=== 14. MASS ASSIGNMENT (role admin harus tetap jadi user) ==="
curl -s -X POST $B/auth/register -H "$J" \
  -d '{"username":"hacker","email":"hacker@example.com","password":"rahasia123","role":"admin"}'
echo ""
echo ""
echo "Cek DB:"
psql -U adityaalifsantoso -d praktikum_backend -c "SELECT username, role FROM users WHERE username='hacker';"

echo ""
echo ""
echo "=========================================="
echo "  SELESAI! Screenshot hasil di atas."
echo "=========================================="