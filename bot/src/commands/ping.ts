// src/commands/ping.ts

import { ChatInputCommandInteraction } from 'discord.js';

export async function handlePing(interaction: ChatInputCommandInteraction): Promise<void> {
  await interaction.reply('Pong!');
}
