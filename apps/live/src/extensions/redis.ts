/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Redis as HocuspocusRedis } from "@hocuspocus/extension-redis";
import { OutgoingMessage } from "@hocuspocus/server";
import type { onConfigurePayload } from "@hocuspocus/server";
import { logger } from "@plane/logger";
import { AppError } from "@/lib/errors";
import { env } from "@/env";
import { getRedisConfiguration } from "./redis-configuration";
import { AdminCommand, adminCommandSchema } from "@/types/admin-commands";
import type { AdminCommandData, AdminCommandHandlers } from "@/types/admin-commands";

export class Redis extends HocuspocusRedis {
  private adminHandlers: AdminCommandHandlers = {};
  private readonly ADMIN_CHANNEL = "hocuspocus:admin";
  private readonly adminSubscriber: HocuspocusRedis["sub"];

  constructor() {
    super(getRedisConfiguration({ url: env.REDIS_URL, host: env.REDIS_HOST, port: env.REDIS_PORT }));
    // Admin commands are JSON, while Hocuspocus decodes every message on its
    // own subscriber as binary collaboration data.
    this.adminSubscriber = this.sub.duplicate();
  }

  async onConfigure(payload: onConfigurePayload) {
    await super.onConfigure(payload);

    // Subscribe to admin channel
    try {
      await this.adminSubscriber.subscribe(this.ADMIN_CHANNEL);
      logger.info(`[Redis] Subscribed to admin channel: ${this.ADMIN_CHANNEL}`);
    } catch (error) {
      logger.error(`[Redis] Failed to subscribe to admin channel:`, error);
      throw error;
    }

    // Listen for admin messages
    this.adminSubscriber.on("message", this.onAdminMessage);
    logger.info(`[Redis] Attached admin message listener`);
  }

  private readonly onAdminMessage = (channel: string, message: string): void => {
    void this.handleAdminMessage(channel, message);
  };

  private handleAdminMessage = async (channel: string, message: string) => {
    if (channel !== this.ADMIN_CHANNEL) return;

    try {
      const parsed = adminCommandSchema.safeParse(JSON.parse(message));
      if (!parsed.success) {
        logger.warn("[Redis] Invalid admin command received", parsed.error);
        return;
      }
      const data = parsed.data;
      if (!this.adminHandlers[data.command]) {
        logger.warn(`[Redis] No handler registered for admin command: ${data.command}`);
        return;
      }
      switch (data.command) {
        case AdminCommand.FORCE_CLOSE:
          await this.adminHandlers[AdminCommand.FORCE_CLOSE]?.(data);
          break;
        case AdminCommand.HEALTH_CHECK:
          await this.adminHandlers[AdminCommand.HEALTH_CHECK]?.(data);
          break;
      }
    } catch (error) {
      logger.error("[Redis] Error handling admin message:", error);
    }
  };

  /**
   * Register handler for an admin command
   */
  public onAdminCommand<Command extends AdminCommandData["command"]>(
    command: Command,
    handler: NonNullable<AdminCommandHandlers[Command]>
  ) {
    this.adminHandlers[command] = handler;
    logger.info(`[Redis] Registered admin command: ${command}`);
  }

  /**
   * Publish admin command to global channel
   */
  public async publishAdminCommand(data: AdminCommandData): Promise<number> {
    const validated = adminCommandSchema.safeParse(data);
    if (!validated.success) throw new AppError(validated.error);
    const message = JSON.stringify(validated.data);
    const receivers = await this.pub.publish(this.ADMIN_CHANNEL, message);

    logger.info(`[Redis] Published "${data.command}" command, received by ${receivers} server(s)`);
    return receivers;
  }

  async onDestroy() {
    // Unsubscribe from admin channel
    try {
      await this.adminSubscriber.unsubscribe(this.ADMIN_CHANNEL);
    } catch (error) {
      logger.error(`[Redis] Error unsubscribing from admin channel:`, error);
    }

    // Remove the message listener to prevent memory leaks
    this.adminSubscriber.removeListener("message", this.onAdminMessage);
    this.adminSubscriber.disconnect();
    logger.info(`[Redis] Removed admin message listener`);

    await super.onDestroy();
  }

  /**
   * Broadcast a message to a document across all servers via Redis.
   * Uses empty identifier so ALL servers process the message.
   */
  public async broadcastToDocument(documentName: string, payload: unknown): Promise<number> {
    const stringPayload = typeof payload === "string" ? payload : JSON.stringify(payload);

    const message = new OutgoingMessage(documentName).writeBroadcastStateless(stringPayload);

    const emptyPrefix = Buffer.concat([Buffer.from([0])]);
    const channel = this["pubKey"](documentName);
    const encodedMessage = Buffer.concat([emptyPrefix, Buffer.from(message.toUint8Array())]);

    const result = await this.pub.publish(channel, encodedMessage);

    logger.info(`REDIS_EXTENSION: Published to ${documentName}, ${result} subscribers`);

    return result;
  }
}
