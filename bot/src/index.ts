// src/index.ts

import { Client, IntentsBitField, Interaction } from 'discord.js';
import { config } from './config';
import { logger } from './logger';
import { handlePing } from './commands/ping';
import { handleSheets } from './commands/sheets';
import { handleSheetsRefresh, SHEETS_REFRESH_PREFIX } from './components/sheetsRefreshButton';

const client = new Client({
  intents: [IntentsBitField.Flags.Guilds],
});

client.once('ready', () => {
  logger.info(`Logged in as ${client.user?.tag}`);
});

client.on('interactionCreate', async (interaction: Interaction) => {
  // Button interactions
  if (interaction.isButton()) {
    if (interaction.customId.startsWith(SHEETS_REFRESH_PREFIX)) {
      await handleSheetsRefresh(interaction);
    }
    return;
  }

  if (!interaction.isChatInputCommand()) return;

  const { commandName } = interaction;
  const ctx = {
    command: commandName,
    userId: interaction.user.id,
    guildId: interaction.guildId ?? undefined,
  };

  switch (commandName) {
    case 'ping':
      await handlePing(interaction);
      break;
    case 'sheets':
      await handleSheets(interaction);
      break;
    default:
      logger.warn('Unknown command received', ctx);
      await interaction.reply({ content: 'Unknown command.', ephemeral: true });
  }
});

client.login(config.botToken);
