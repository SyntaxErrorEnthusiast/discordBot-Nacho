# discordBot

A Discord bot built with [discord.js](https://discord.js.org/) and TypeScript.

## Commands

| Command | Description |
|---|---|
| `/ping` | Replies with Pong! |
| `/sheets <url>` | Screenshots a public Google Sheet and posts it |

## Prerequisites

- [Node.js](https://nodejs.org/) v18+
- A Discord application and bot token from the [Discord Developer Portal](https://discord.com/developers/applications)
- A Google Cloud API key with the Sheets API enabled (for `/sheets`)

## Setup

**1. Install dependencies**
```bash
cd bot
npm install
```

**2. Configure environment variables**

Create a `.env` file inside the `bot/` directory:
```env
BOT_TOKEN=your_discord_bot_token
CLIENT_ID=your_discord_application_client_id
SHEETS_KEY=your_google_sheets_api_key
STEAM_API_KEY=your_steam_api_key
GUILD_ID=your_guild_id   # optional — omit to deploy commands globally
```

**3. Deploy slash commands**
```bash
npm run deploy-commands
```

**4. Build and start the bot**
```bash
npm run build
npm start
```

## Project structure

```
bot/
  src/
    commands/        # Thin Discord interaction handlers
    services/        # Business logic (e.g. Google Sheets screenshotter)
    config.ts        # Centralised, validated environment config
    logger.ts        # Structured JSON logger
    deploy-commands.ts
    index.ts         # Entry point — wires up the Discord client
```
