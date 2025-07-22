#!/bin/bash

# One-time Railway setup script
# Usage: ./setup-railway.sh (run once to link project)

set -e

echo "🔗 Setting up Railway project link..."

# Check if Railway CLI is installed
if ! command -v railway &> /dev/null; then
    echo "❌ Railway CLI not found. Installing..."
    npm install -g @railway/cli
fi

# Check if logged in to Railway
if ! railway whoami &> /dev/null; then
    echo "🔐 Please login to Railway..."
    railway login
fi

# Link to existing egobot project
echo "🔗 Linking to egobot project..."
railway link

echo "✅ Railway project linked successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Set environment variables in Railway dashboard:"
echo "   railway open"
echo ""
echo "2. Deploy the application:"
echo "   ./deploy.sh" 