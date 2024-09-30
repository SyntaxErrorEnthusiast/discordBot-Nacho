// src/index.ts

import { Client, IntentsBitField, Interaction } from 'discord.js';
import { config } from 'dotenv';

config(); // Load environment variables from .env file

const token = process.env.BOT_TOKEN;

const client = new Client({
  intents: [IntentsBitField.Flags.Guilds],
});

client.once('ready', () => {
  console.log(`Logged in as ${client.user?.tag}!`);
});

client.on('interactionCreate', async (interaction: Interaction) => {
  if (!interaction.isChatInputCommand()) return;

  const { commandName } = interaction;

  switch (commandName) {
    case 'ping':
      await interaction.reply('Pong!');
      break;
    case 'echo':
      var message = interaction.options.getString('message');
      await interaction.reply(message || 'No message provided.');
      break;
    case 'bitch':
      var message = interaction.options.getString('message');
      await interaction.reply(message || 'No message provided.');
      break;
    default:
     interaction.reply('Invalid Command');
    }
});

client.login(token);
