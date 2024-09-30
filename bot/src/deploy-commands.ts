// src/deploy-commands.ts

import { REST, Routes, SlashCommandBuilder } from 'discord.js';
import { config } from 'dotenv';

config();

const token = process.env.BOT_TOKEN;
const clientId = process.env.CLIENT_ID;
const guildId = process.env.GUILD_ID; // Optional: For guild-specific commands

const commands = [
  new SlashCommandBuilder()
    .setName('ping')
    .setDescription('Replies with Pong!'),
  new SlashCommandBuilder()
    .setName('echo')
    .setDescription('Replies with your input!')
    .addStringOption((option) =>
      option.setName('message').setDescription('The message to echo').setRequired(true)
    ),
  new SlashCommandBuilder()
    .setName('bitch')
    .setDescription('Replies with your input!')
    .addStringOption((option) =>
      option.setName('message').setDescription('The message to bitch').setRequired(true)
    ),
].map((command) => command.toJSON());

const rest = new REST({ version: '10' }).setToken(token!);

(async () => {
  try {
    console.log('Started refreshing application (/) commands.');

    if (guildId) {
      // For guild-specific commands
      await rest.put(Routes.applicationGuildCommands(clientId!, guildId), { body: commands });
    } else {
      // For global commands
      await rest.put(Routes.applicationCommands(clientId!), { body: commands });
    }

    console.log('Successfully reloaded application (/) commands.');
  } catch (error) {
    console.error(error);
  }
})();
