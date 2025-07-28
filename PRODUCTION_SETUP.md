# Production Setup Guide - Railway Deployment

## 🚀 Railway Production Deployment

Your Railway cron job is now configured to run every morning at 6:00 AM CET with the command `./processor -once`.

### **📋 Required Environment Variables**

Set these in your Railway dashboard under **Variables**:

#### **🔑 OpenAI Configuration**
```bash
OPENAI_API_KEY=sk-your-openai-api-key-here
OPENAI_STUB=false
```

#### **📧 IMAP Email Configuration (Gmail)**
```bash
IMAP_USERNAME=your-email@gmail.com
IMAP_PASSWORD=your-16-char-app-password
IMAP_SERVER=imap.gmail.com
IMAP_PORT=993
IMAP_FOLDER=INBOX
```

#### **📤 SMTP Email Configuration (Gmail)**
```bash
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-16-char-app-password
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_FROM=your-email@gmail.com
SMTP_TO=your-email@gmail.com
```

#### **⚙️ Processing Configuration**
```bash
ENTITIES_TO_TRACK=["Benny Gotfred Schmidt","0605410146","Lægårdsvej 12A"]
SCHEDULE_CRON=0 6 * * *
MAX_RETRIES=3
RETRY_DELAY=5m
```

### **🔧 Gmail App Password Setup**

1. **Enable 2-Factor Authentication**:
   - Go to [Google Account Security](https://myaccount.google.com/security)
   - Enable "2-Step Verification"

2. **Generate App Passwords**:
   - Go to [App Passwords](https://myaccount.google.com/apppasswords)
   - Generate two 16-character passwords:
     - **IMAP**: Select "Mail" → "Other (Custom name)" → Name it "egobot-imap"
     - **SMTP**: Select "Mail" → "Other (Custom name)" → Name it "egobot-smtp"

3. **Use the 16-character passwords** in your environment variables (not your regular Gmail password)

### **⏰ Cron Job Configuration**

The Railway cron job is configured in `railway.json`:
```json
{
  "cron": {
    "schedule": "0 6 * * *",
    "command": "./processor -once"
  }
}
```

- **Schedule**: `0 6 * * *` = Daily at 6:00 AM CET
- **Command**: `./processor -once` = Run once and exit
- **Timezone**: Railway uses UTC, so 6:00 AM CET = 5:00 AM UTC in winter, 4:00 AM UTC in summer

### **🔍 Verification Steps**

1. **Check Railway Dashboard**:
   - Go to your Railway project dashboard
   - Check that all environment variables are set
   - Verify the cron job is enabled

2. **Test the Setup**:
   - Railway will automatically deploy when you push to GitHub
   - The cron job will run at 6:00 AM CET daily
   - Check the Railway logs to see if the job runs successfully

3. **Monitor Logs**:
   - In Railway dashboard, go to **Deployments** → **Logs**
   - Look for cron job execution logs
   - Check for any error messages

### **📊 Expected Behavior**

**Every morning at 6:00 AM CET**:
1. ✅ **Cron job triggers** `./processor -once`
2. ✅ **Connects to Gmail** via IMAP
3. ✅ **Downloads latest PDF** from emails (last 24 hours)
4. ✅ **Analyzes PDF** with your target entities
5. ✅ **Sends email report** to your inbox
6. ✅ **Exits cleanly** after processing

### **🔧 Troubleshooting**

**Cron job not running**:
- Check Railway dashboard → **Cron** tab
- Verify environment variables are set
- Check Railway logs for errors

**Email connection issues**:
- Verify App Passwords are correct (16 characters)
- Check 2-Factor Authentication is enabled
- Test with `go run test_smtp.go` locally

**No PDFs found**:
- Check emails are from last 24 hours
- Verify PDFs are actual attachments (not embedded)
- Check IMAP folder setting

**OpenAI API errors**:
- Verify `OPENAI_API_KEY` is set correctly
- Check `OPENAI_STUB=false` is set
- Monitor token usage in OpenAI dashboard

### **💰 Cost Estimation**

**Railway Free Tier**:
- ✅ **500 hours/month** (free)
- ✅ **Daily cron job**: ~1 minute/day = 30 hours/month
- ✅ **Well within limits**

**OpenAI API Costs**:
- ✅ **GPT-3.5-turbo**: ~$0.002 per 1K tokens
- ✅ **Typical PDF**: ~$0.01-0.05 per analysis
- ✅ **Daily cost**: ~$0.30-1.50/month

### **🎯 Success Indicators**

You'll know it's working when:
- ✅ **Daily emails** arrive in your inbox at ~6:00 AM CET
- ✅ **Email contains** analysis results for your entities
- ✅ **Railway logs** show successful cron job execution
- ✅ **No errors** in Railway or OpenAI dashboards

### **📱 Morning Coffee Setup**

Your perfect morning routine:
1. ☕ **6:00 AM**: Cron job runs automatically
2. 📧 **6:01 AM**: Analysis email arrives in your inbox
3. 📖 **6:05 AM**: Read fresh Statstidende analysis with coffee
4. 🎯 **6:10 AM**: Take action on any relevant findings

**Happy analyzing! 🚀** 