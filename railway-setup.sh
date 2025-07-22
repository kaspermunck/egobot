#!/bin/bash

# Railway Setup Script for egobot
# Usage: ./railway-setup.sh

set -e

echo "🚀 Setting up Railway for egobot..."

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

# Initialize Railway project (this creates the .railway directory)
echo "📁 Initializing Railway project..."
if [ ! -d ".railway" ]; then
    railway init
    echo "✅ Railway project initialized"
else
    echo "✅ Railway project already exists"
fi

# Link to existing project or create new one
echo "🔗 Linking to Railway project..."
railway link

echo "✅ Railway setup complete!"
echo ""
echo "📋 Next steps:"
echo "1. Set environment variables in Railway dashboard:"
echo "   railway open"
echo ""
echo "2. Deploy the application:"
echo "   railway up"
echo ""
echo "3. View logs:"
echo "   railway logs"
echo ""
echo "4. Open dashboard:"
echo "   railway open" 