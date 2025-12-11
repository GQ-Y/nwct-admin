#!/bin/bash

# API测试脚本

BASE_URL="http://localhost:8080/api/v1"

echo "=== 测试初始化状态 ==="
curl -s "$BASE_URL/config/init/status" | python3 -m json.tool || curl -s "$BASE_URL/config/init/status"
echo -e "\n"

echo "=== 测试登录 ==="
TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "Token: $TOKEN"
echo -e "\n"

if [ -z "$TOKEN" ]; then
    echo "登录失败，退出测试"
    exit 1
fi

echo "=== 测试系统信息 ==="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/system/info" | python3 -m json.tool || curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/system/info"
echo -e "\n"

echo "=== 测试网络接口列表 ==="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/network/interfaces" | python3 -m json.tool || curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/network/interfaces"
echo -e "\n"

echo "=== 测试网络状态 ==="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/network/status" | python3 -m json.tool || curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/network/status"
echo -e "\n"

echo "=== 测试DNS查询 ==="
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"google.com","type":"A"}' \
  "$BASE_URL/tools/dns" | python3 -m json.tool || curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"query":"google.com","type":"A"}' "$BASE_URL/tools/dns"
echo -e "\n"

echo "=== 测试Ping ==="
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"target":"8.8.8.8","count":3}' \
  "$BASE_URL/tools/ping" | python3 -m json.tool || curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"target":"8.8.8.8","count":3}' "$BASE_URL/tools/ping"
echo -e "\n"

echo "=== 测试设备列表 ==="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/devices" | python3 -m json.tool || curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/devices"
echo -e "\n"

echo "=== 测试MQTT状态 ==="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/mqtt/status" | python3 -m json.tool || curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/mqtt/status"
echo -e "\n"

echo "=== 测试NPS状态 ==="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/nps/status" | python3 -m json.tool || curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/nps/status"
echo -e "\n"

echo "测试完成！"

