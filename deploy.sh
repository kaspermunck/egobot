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

# Create new project if it doesn't exist
if ! railway project &> /dev/null; then
    echo "📁 Creating new Railway project..."
    railway init
fi

# Set environment variables (you'll need to set these in Railway dashboard)
echo "⚙️  Setting up environment variables..."
echo "Please set the following environment variables in Railway dashboard:"
echo ""
echo "Required variables:"
echo "- IMAP_USERNAME=your-email@gmail.com"
echo "- IMAP_PASSWORD=your-app-password"
echo "- SMTP_USERNAME=your-email@gmail.com"
echo "- SMTP_PASSWORD=your-app-password"
echo "- SMTP_FROM=your-email@gmail.com"
echo "- SMTP_TO=recipient@example.com"
echo "- OPENAI_API_KEY=your-openai-key"
echo "- ENTITIES_TO_TRACK=[\"Danske Bank\", \"fintech\", \"bankruptcy\"]"
echo ""
echo "Optional variables:"
echo "- OPENAI_STUB=false (set to true for testing)"
echo "- SCHEDULE_CRON=0 6 * * * (daily at 6am CET)"
echo ""

# Deploy to Railway
echo "🚀 Deploying to Railway..."
railway up

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