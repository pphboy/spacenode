#!/bin/bash
# Test GET /space/list
curl -X GET http://localhost:8080/space/list -H "X-Hc-User-Id: dzh"

# Test GET /app/list
curl -X GET http://localhost:8080/app/list -H "X-Hc-User-Id: dzh"

# Test POST /app/add
curl -X POST http://localhost:8080/app/add?appid=test-app -H "X-Hc-User-Id: dzh"

# Test POST /app/remove
curl -X POST http://localhost:8080/app/remove?appid=test-app -H "X-Hc-User-Id: dzh"

# Test GET /lzcapp/applist
curl -X GET http://localhost:8080/lzcapp/applist -H "X-Hc-User-Id: dzh"
