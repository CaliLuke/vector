/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { z } from "zod";

/**
 * Type-safe admin commands for server-to-server communication
 */

/**
 * Force close error codes - reasons why a document is being force closed
 */
export enum ForceCloseReason {
  CRITICAL_ERROR = "critical_error",
  MEMORY_LEAK = "memory_leak",
  DOCUMENT_TOO_LARGE = "document_too_large",
  ADMIN_REQUEST = "admin_request",
  SERVER_SHUTDOWN = "server_shutdown",
  SECURITY_VIOLATION = "security_violation",
  CORRUPTION_DETECTED = "corruption_detected",
}

/**
 * WebSocket close codes
 * https://developer.mozilla.org/en-US/docs/Web/API/CloseEvent/code
 */
export enum CloseCode {
  /** Normal closure; the connection successfully completed */
  NORMAL = 1000,
  /** The endpoint is going away (server shutdown or browser navigating away) */
  GOING_AWAY = 1001,
  /** Protocol error */
  PROTOCOL_ERROR = 1002,
  /** Unsupported data */
  UNSUPPORTED_DATA = 1003,
  /** Reserved (no status code was present) */
  NO_STATUS = 1005,
  /** Abnormal closure */
  ABNORMAL = 1006,
  /** Invalid frame payload data */
  INVALID_DATA = 1007,
  /** Policy violation */
  POLICY_VIOLATION = 1008,
  /** Message too big */
  MESSAGE_TOO_BIG = 1009,
  /** Client expected extension not negotiated */
  MANDATORY_EXTENSION = 1010,
  /** Server encountered unexpected condition */
  INTERNAL_ERROR = 1011,
  /** Custom: Force close requested */
  FORCE_CLOSE = 4000,
  /** Custom: Document too large */
  DOCUMENT_TOO_LARGE = 4001,
  /** Custom: Memory pressure */
  MEMORY_PRESSURE = 4002,
  /** Custom: Security violation */
  SECURITY_VIOLATION = 4003,
}

/**
 * Admin command types
 */
export enum AdminCommand {
  FORCE_CLOSE = "force_close",
  HEALTH_CHECK = "health_check",
}

/**
 * Force close command data structure
 */
const forceCloseCommandSchema = z.object({
  command: z.literal(AdminCommand.FORCE_CLOSE),
  docId: z.string().min(1),
  reason: z.enum(ForceCloseReason),
  code: z.enum(CloseCode),
  originServer: z.string().min(1),
  timestamp: z.string().optional(),
});
const healthCheckCommandSchema = z.object({
  command: z.literal(AdminCommand.HEALTH_CHECK),
  originServer: z.string().min(1),
  timestamp: z.string(),
});
export const adminCommandSchema = z.discriminatedUnion("command", [forceCloseCommandSchema, healthCheckCommandSchema]);
export type ForceCloseCommandData = z.infer<typeof forceCloseCommandSchema>;
export type HealthCheckCommandData = z.infer<typeof healthCheckCommandSchema>;
export type AdminCommandData = z.infer<typeof adminCommandSchema>;
export type AdminCommandHandlers = {
  [Command in AdminCommandData["command"]]?: AdminCommandHandler<Extract<AdminCommandData, { command: Command }>>;
};

/**
 * Client force close message structure (sent to clients via sendStateless)
 */
export interface ClientForceCloseMessage {
  type: "force_close";
  reason: ForceCloseReason;
  code: CloseCode;
  message?: string;
  timestamp?: string;
}

/**
 * Admin command handler function type
 */
type AdminCommandHandler<T extends AdminCommandData = AdminCommandData> = (data: T) => Promise<void> | void;

/**
 * Type guard to check if data is a ForceCloseCommandData
 */
export function isForceCloseCommand(data: AdminCommandData): data is ForceCloseCommandData {
  return data.command === AdminCommand.FORCE_CLOSE;
}

/**
 * Get human-readable message for force close reason
 */
export function getForceCloseMessage(reason: ForceCloseReason): string {
  const messages: Record<ForceCloseReason, string> = {
    [ForceCloseReason.CRITICAL_ERROR]: "A critical error occurred. Please refresh the page.",
    [ForceCloseReason.MEMORY_LEAK]: "Memory limit exceeded. Please refresh the page.",
    [ForceCloseReason.DOCUMENT_TOO_LARGE]:
      "Content limit reached and live sync is off. Create a new page or use nested pages to continue syncing.",
    [ForceCloseReason.ADMIN_REQUEST]: "Connection closed by administrator. Please try again later.",
    [ForceCloseReason.SERVER_SHUTDOWN]: "Server is shutting down. Please reconnect in a moment.",
    [ForceCloseReason.SECURITY_VIOLATION]: "Security violation detected. Connection terminated.",
    [ForceCloseReason.CORRUPTION_DETECTED]: "Data corruption detected. Please refresh the page.",
  };

  return messages[reason] || "Connection closed. Please refresh the page.";
}
