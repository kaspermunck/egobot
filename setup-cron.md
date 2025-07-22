# Railway Cron Job Setup

## Automatic Setup (Recommended)

The `railway.json` file now includes cron configuration that should be automatically detected by Railway.

## Manual Setup (if automatic doesn't work)

1. **Open Railway Dashboard:**
   ```bash
   railway open
   ```

2. **Go to your egobot project**

3. **Navigate to "Settings" tab**

4. **Find "Cron Jobs" section**

5. **Add new cron job:**
   - **Schedule:** `0 6 * * *` (Daily at 6:00 AM CET)
   - **Command:** `./processor -once`
   - **Description:** `Daily PDF analysis`

## Verify Cron Job

1. **Check cron job status:**
   ```bash
   railway logs
   ```

2. **Test manually:**
   ```bash
   railway run ./processor -once
   ```

## Cron Schedule Format

- `0 6 * * *` = Daily at 6:00 AM
- `0 9 * * *` = Daily at 9:00 AM  
- `0 6 * * 1-5` = Weekdays at 6:00 AM
- `0 6 1 * *` = Monthly on 1st at 6:00 AM

## Environment Variables for Cron

Make sure these are set in Railway dashboard:
- `IMAP_USERNAME`
- `IMAP_PASSWORD` 
- `SMTP_USERNAME`
- `SMTP_PASSWORD`
- `SMTP_FROM`
- `SMTP_TO`
- `OPENAI_API_KEY`
- `ENTITIES_TO_TRACK` 