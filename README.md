# discordBot

A Discord bot built with [discord.js](https://discord.js.org/) and TypeScript.

## Commands

| Command | Description |
|---|---|
| `/sheets <url>` | Screenshots a public Google Sheet and posts it, with a Refresh button to re-fetch on demand |

## Prerequisites

- A Discord application and bot token from the [Discord Developer Portal](https://discord.com/developers/applications)
- A Google Cloud API key with the Sheets API enabled

## Running with Docker (recommended)

A Docker image is automatically built and published to GitHub Container Registry on every push to `main`.

**1. Pull the image**
```bash
docker pull ghcr.io/syntaxerrorenthusiast/discordbot-nacho:latest
```

**2. Create a `.env` file**
```env
BOT_TOKEN=your_discord_bot_token
CLIENT_ID=your_discord_application_client_id
SHEETS_KEY=your_google_sheets_api_key
GUILD_ID=your_guild_id   # optional — omit to deploy commands globally
```

**3. Run the bot**
```bash
docker run --env-file .env ghcr.io/syntaxerrorenthusiast/discordbot-nacho:latest
```

## Running locally

Requires [Node.js](https://nodejs.org/) v18+.

**1. Install dependencies**
```bash
cd bot
npm install
```

**2. Create a `.env` file in the `bot/` directory** (same variables as above)

**3. Deploy slash commands**
```bash
npm run deploy-commands
```

**4. Build and start**
```bash
npm run build
npm start
```

## Project structure

```
bot/
  src/
    commands/          # Thin Discord interaction handlers
    components/        # Message component handlers (buttons)
    services/          # Business logic (Google Sheets screenshotter)
    config.ts          # Centralised, validated environment config
    logger.ts          # Structured JSON logger
    deploy-commands.ts # Registers slash commands with Discord
    index.ts           # Entry point — wires up the Discord client
```
