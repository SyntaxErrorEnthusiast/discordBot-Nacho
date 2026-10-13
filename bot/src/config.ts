// src/config.ts
// Centralized configuration — validated once at startup.
// All other modules import from here rather than reading process.env directly.

import { config as loadEnv } from 'dotenv';

loadEnv();

function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

export const config = {
  botToken: requireEnv('BOT_TOKEN'),
  steamApiKey: requireEnv('STEAM_API_KEY'),
  sheetsApiKey: requireEnv('SHEETS_KEY'),
  clientId: requireEnv('CLIENT_ID'),
  // GUILD_ID is optional — omitting it deploys commands globally
  guildId: process.env.GUILD_ID,
} as const;
