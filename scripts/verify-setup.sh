#!/bin/bash

echo "🔍 Verifying Autonomous Task Management Setup..."
echo ""

# Check Memos
echo -n "Checking Memos... "
if curl -s http://localhost:5230 > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Not accessible"
fi

# Check Qdrant
echo -n "Checking Qdrant... "
if curl -s http://localhost:6333/health > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Not accessible"
fi

# Check Backend
echo -n "Checking Backend... "
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Not accessible"
fi

echo ""
echo "🎉 Setup verification complete!"
