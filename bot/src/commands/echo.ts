// src/commands/echo.ts

import { ChatInputCommandInteraction } from 'discord.js';

export async function handleEcho(interaction: ChatInputCommandInteraction): Promise<void> {
  const message = interaction.options.getString('message');
  await interaction.reply(message ?? 'No message provided.');
}
