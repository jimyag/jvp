#!/bin/bash

echo "🚀 Starting JVP Admin Dashboard..."
echo ""

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
    echo ""
fi

# Check if backend is running
echo "🔍 Checking if JVP backend is running..."
if curl -s http://192.168.2.100:8080/api > /dev/null 2>&1; then
    echo "✅ Backend is running on http://192.168.2.100:8080"
else
    echo "⚠️  Warning: Backend not detected on http://192.168.2.100:8080"
    echo "   Make sure to start your JVP backend before using the dashboard"
fi

echo ""
echo "🌐 Starting development server..."
echo "   Dashboard will be available at http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

npm run dev
