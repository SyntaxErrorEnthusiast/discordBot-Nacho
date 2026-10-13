// src/logger.ts
// Centralized structured logger. All modules use this instead of console directly.
// Outputs JSON lines to stdout so logs are machine-readable in production.

type LogLevel = 'info' | 'warn' | 'error';

interface LogContext {
  command?: string;
  userId?: string;
  guildId?: string;
  [key: string]: unknown;
}

function write(level: LogLevel, message: string, context?: LogContext, error?: unknown): void {
  const entry: Record<string, unknown> = {
    timestamp: new Date().toISOString(),
    level,
    message,
    ...context,
  };

  if (error instanceof Error) {
    entry.error = error.message;
    entry.stack = error.stack;
  } else if (error !== undefined) {
    entry.error = String(error);
  }

  console.log(JSON.stringify(entry));
}

export const logger = {
  info: (message: string, context?: LogContext) => write('info', message, context),
  warn: (message: string, context?: LogContext) => write('warn', message, context),
  error: (message: string, context?: LogContext, error?: unknown) => write('error', message, context, error),
};
