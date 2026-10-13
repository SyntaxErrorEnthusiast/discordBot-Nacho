// src/services/sheetsService.ts
// Business logic for fetching and rasterising public Google Sheets.

import { google } from 'googleapis';
import { pdf } from 'pdf-to-img';
import sharp from 'sharp';
import { logger } from '../logger';

export class SheetNotPublicError extends Error {
  constructor() {
    super('Sheet is not public or could not be accessed.');
    this.name = 'SheetNotPublicError';
  }
}

/**
 * Parses a Google Sheets URL and extracts the spreadsheet ID and optional gid.
 * Throws if the URL is not a recognisable Sheets URL.
 */
function parseSheetUrl(url: string): { spreadsheetId: string; gid: number } {
  const idMatch = url.match(/\/spreadsheets\/d\/([a-zA-Z0-9_-]+)/);
  if (!idMatch) {
    throw new Error('Could not parse a spreadsheet ID from the provided URL.');
  }
  const spreadsheetId = idMatch[1];

  const gidMatch = url.match(/[#&?]gid=(\d+)/);
  const gid = gidMatch ? parseInt(gidMatch[1], 10) : 0;

  return { spreadsheetId, gid };
}

/**
 * Returns the A1 notation bounding box of non-empty cells, e.g. "A1:F20".
 * Uses only cell values for trimming — styled-but-empty cells are not counted.
 */
async function getUsedRange(spreadsheetId: string, gid: number, apiKey: string): Promise<string> {
  const sheets = google.sheets({ version: 'v4' });

  const metaRes = await sheets.spreadsheets.get({
    spreadsheetId,
    key: apiKey,
    fields: 'sheets.properties(sheetId,title)',
  });

  const sheetMeta = metaRes.data.sheets?.find((s) => s.properties?.sheetId === gid);
  if (!sheetMeta?.properties?.title) {
    throw new Error(`No sheet found with gid ${gid}.`);
  }
  const title = sheetMeta.properties.title;

  const valuesRes = await sheets.spreadsheets.values.get({
    spreadsheetId,
    key: apiKey,
    range: `'${title.replace(/'/g, "''")}'`,
    valueRenderOption: 'FORMATTED_VALUE',
  });

  const rows = valuesRes.data.values ?? [];
  let top = Infinity, bottom = -1, left = Infinity, right = -1;

  rows.forEach((row, r) => {
    row.forEach((cell, c) => {
      if (cell !== null && cell !== undefined && String(cell).trim() !== '') {
        top = Math.min(top, r);
        bottom = Math.max(bottom, r);
        left = Math.min(left, c);
        right = Math.max(right, c);
      }
    });
  });

  if (bottom < 0) {
    throw new Error('The sheet appears to be empty.');
  }

  return `${colToLetter(left)}${top + 1}:${colToLetter(right)}${bottom + 1}`;
}

function colToLetter(index: number): string {
  let n = index + 1;
  let s = '';
  while (n > 0) {
    const r = (n - 1) % 26;
    s = String.fromCharCode(65 + r) + s;
    n = Math.floor((n - 1) / 26);
  }
  return s;
}

/**
 * Exports the given range from a public sheet as a PDF buffer.
 * Throws SheetNotPublicError if Google redirects to a login page.
 */
async function exportAsPdf(spreadsheetId: string, gid: number, range: string): Promise<Buffer> {
  const params = new URLSearchParams({
    format: 'pdf',
    gid: String(gid),
    range,
    size: 'A4',
    portrait: 'false',
    scale: '4',
    gridlines: 'false',
    printtitle: 'false',
    sheetnames: 'false',
    pagenum: 'UNDEFINED',
    fzr: 'false',
    top_margin: '0',
    bottom_margin: '0',
    left_margin: '0',
    right_margin: '0',
  });

  const url = `https://docs.google.com/spreadsheets/d/${spreadsheetId}/export?${params}`;
  const res = await fetch(url, { redirect: 'follow' });

  // Google redirects to accounts.google.com when the sheet requires auth
  if (res.url.includes('accounts.google.com') || res.status === 401 || res.status === 403) {
    throw new SheetNotPublicError();
  }

  if (!res.ok) {
    throw new Error(`PDF export request failed with status ${res.status}.`);
  }

  return Buffer.from(await res.arrayBuffer());
}

/**
 * Rasterizes the first page of a PDF buffer to a trimmed PNG buffer.
 */
async function pdfToTrimmedPng(pdfBuf: Buffer): Promise<Buffer> {
  const doc = await pdf(pdfBuf, { scale: 3 });

  if (doc.length > 1) {
    logger.warn('Sheet export produced multiple pages; only page 1 will be shown.');
  }

  const firstPage = await doc.getPage(1);

  return sharp(firstPage)
    .trim({ background: '#ffffff', threshold: 10 })
    .extend({ top: 12, bottom: 12, left: 12, right: 12, background: '#ffffff' })
    .png()
    .toBuffer();
}

/**
 * Takes a public Google Sheets URL and returns a PNG buffer of the sheet's
 * non-empty content area, preserving all Google-rendered formatting.
 *
 * Throws SheetNotPublicError if the sheet is private or inaccessible.
 */
export async function screenshotSheet(url: string, apiKey: string): Promise<Buffer> {
  const { spreadsheetId, gid } = parseSheetUrl(url);
  const range = await getUsedRange(spreadsheetId, gid, apiKey);
  const pdfBuf = await exportAsPdf(spreadsheetId, gid, range);
  return pdfToTrimmedPng(pdfBuf);
}
