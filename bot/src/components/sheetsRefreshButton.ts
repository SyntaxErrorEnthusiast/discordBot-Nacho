// src/components/sheetsRefreshButton.ts

import { createHash } from 'crypto';
import { AttachmentBuilder, ButtonBuilder, ButtonStyle, ButtonInteraction } from 'discord.js';
import { screenshotSheet, SheetNotPublicError } from '../services/sheetsService';
import { config } from '../config';
import { logger } from '../logger';

export const SHEETS_REFRESH_PREFIX = 'sheets_refresh:';

// In-memory map from short hash -> original URL.
// Entries live as long as the bot process does, which is fine for a simple bot.
const urlStore = new Map<string, string>();

function storeUrl(url: string): string {
  const hash = createHash('sha1').update(url).digest('hex').slice(0, 16);
  urlStore.set(hash, url);
  return hash;
}

/**
 * Builds the Refresh button for a given sheet URL.
 * Stores the URL in memory and uses a short hash as the custom ID.
 */
export function buildRefreshButton(url: string): ButtonBuilder {
  const hash = storeUrl(url);
  return new ButtonBuilder()
    .setCustomId(`${SHEETS_REFRESH_PREFIX}${hash}`)
    .setLabel('Refresh')
    .setEmoji('🔄')
    .setStyle(ButtonStyle.Secondary);
}

/**
 * Handles a click on the Refresh button.
 * Looks up the original URL from the hash, re-fetches, and edits the message.
 */
export async function handleSheetsRefresh(interaction: ButtonInteraction): Promise<void> {
  const hash = interaction.customId.slice(SHEETS_REFRESH_PREFIX.length);
  const url = urlStore.get(hash);

  if (!url) {
    await interaction.reply({
      content: 'This button has expired. Please run `/sheets` again.',
      ephemeral: true,
    });
    return;
  }

  const ctx = {
    command: 'sheets_refresh',
    userId: interaction.user.id,
    guildId: interaction.guildId ?? undefined,
  };

  await interaction.deferUpdate();

  try {
    const png = await screenshotSheet(url, config.sheetsApiKey);
    const attachment = new AttachmentBuilder(png, { name: 'sheet.png' });
    await interaction.editReply({ files: [attachment] });
  } catch (err) {
    if (err instanceof SheetNotPublicError) {
      await interaction.editReply({
        content: 'That sheet doesn\'t appear to be public. Make sure sharing is set to "Anyone with the link".',
        files: [],
      });
    } else {
      logger.error('Failed to refresh sheet screenshot', ctx, err);
      await interaction.editReply({
        content: 'Something went wrong while refreshing the screenshot. Please try again.',
        files: [],
      });
    }
  }
}
