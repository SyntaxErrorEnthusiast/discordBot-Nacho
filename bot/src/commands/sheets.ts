// src/commands/sheets.ts

import { AttachmentBuilder, ActionRowBuilder, ButtonBuilder, ChatInputCommandInteraction } from 'discord.js';
import { screenshotSheet, SheetNotPublicError } from '../services/sheetsService';
import { buildRefreshButton } from '../components/sheetsRefreshButton';
import { config } from '../config';
import { logger } from '../logger';

export async function handleSheets(interaction: ChatInputCommandInteraction): Promise<void> {
  const url = interaction.options.getString('url');
  if (!url) {
    await interaction.reply({ content: 'Please provide a Google Sheets URL.', ephemeral: true });
    return;
  }

  // Defer immediately — the export + render takes a few seconds
  await interaction.deferReply();

  const ctx = {
    command: 'sheets',
    userId: interaction.user.id,
    guildId: interaction.guildId ?? undefined,
  };

  try {
    const png = await screenshotSheet(url, config.sheetsApiKey);
    const attachment = new AttachmentBuilder(png, { name: 'sheet.png' });
    const row = new ActionRowBuilder<ButtonBuilder>().addComponents(buildRefreshButton(url));
    await interaction.followUp({ files: [attachment], components: [row] });
  } catch (err) {
    if (err instanceof SheetNotPublicError) {
      await interaction.followUp('That sheet doesn\'t appear to be public. Make sure sharing is set to "Anyone with the link".');
    } else {
      logger.error('Failed to screenshot sheet', ctx, err);
      await interaction.followUp('Something went wrong while generating the screenshot. Please try again.');
    }
  }
}
