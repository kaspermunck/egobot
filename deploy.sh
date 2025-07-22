#!/bin/bash

# Railway Deployment Script for egobot
# Usage: ./deploy.sh

set -e

echo "🚀 Deploying egobot to Railway..."

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

# Deploy to Railway (non-interactive)
echo "🚀 Deploying to Railway..."
railway up --detach

echo "✅ Deployment complete!"
echo ""
echo "📊 Check your deployment:"
echo "railway status"
echo ""
echo "📋 View logs:"
echo "railway logs"
echo ""
echo "🌐 Open dashboard:"
echo "railway open" 