// src/deploy-commands.ts

import { REST, Routes, SlashCommandBuilder } from 'discord.js';
import { config } from './config';

const commands = [
  new SlashCommandBuilder()
    .setName('ping')
    .setDescription('Replies with Pong!'),
  new SlashCommandBuilder()
    .setName('sheets')
    .setDescription('Screenshots a public Google Sheet and posts it here.')
    .addStringOption((option) =>
      option.setName('url').setDescription('The public Google Sheets URL').setRequired(true)
    ),
].map((command) => command.toJSON());

const rest = new REST({ version: '10' }).setToken(config.botToken);

(async () => {
  try {
    console.log('Started refreshing application (/) commands.');

    if (config.guildId) {
      await rest.put(Routes.applicationGuildCommands(config.clientId, config.guildId), { body: commands });
    } else {
      await rest.put(Routes.applicationCommands(config.clientId), { body: commands });
    }

    console.log('Successfully reloaded application (/) commands.');
  } catch (error) {
    console.error(error);
  }
})();
