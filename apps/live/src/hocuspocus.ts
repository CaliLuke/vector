/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Hocuspocus } from "@hocuspocus/server";
import { v4 as uuidv4 } from "uuid";
// env
import { env } from "@/env";
// extensions
import { getExtensions } from "@/extensions";
// lib
import { onAuthenticate } from "@/lib/auth";
import { onStateless } from "@/lib/stateless";

export class HocusPocusServerManager {
  private static instance: HocusPocusServerManager | null = null;
  private server: Hocuspocus | null = null;
  private destroyPromise: Promise<void> | null = null;
  // server options
  private serverName = env.HOSTNAME || uuidv4();

  private constructor() {
    // Private constructor to prevent direct instantiation
  }

  /**
   * Get the singleton instance of HocusPocusServerManager
   */
  public static getInstance(): HocusPocusServerManager {
    if (!HocusPocusServerManager.instance) {
      HocusPocusServerManager.instance = new HocusPocusServerManager();
    }
    return HocusPocusServerManager.instance;
  }

  /**
   * Initialize and configure the HocusPocus server
   */
  public async initialize(): Promise<Hocuspocus> {
    if (this.server) {
      return this.server;
    }

    this.server = new Hocuspocus({
      name: this.serverName,
      onAuthenticate,
      onStateless,
      extensions: getExtensions(),
      debounce: 10000,
    });

    return this.server;
  }

  /**
   * Get the configured server instance
   */
  public getServer(): Hocuspocus | null {
    return this.server;
  }

  public async destroy(): Promise<void> {
    if (this.destroyPromise) return this.destroyPromise;

    this.destroyPromise = this.runDestroy();
    try {
      await this.destroyPromise;
    } finally {
      this.destroyPromise = null;
    }
  }

  private async runDestroy(): Promise<void> {
    const server = this.server;
    if (!server) return;

    await new Promise<void>((resolve) => {
      server.configuration.extensions.push({
        async afterUnloadDocument({ instance }) {
          if (instance.getDocumentsCount() === 0) resolve();
        },
      });

      if (server.getDocumentsCount() === 0) resolve();
      server.closeConnections();
      server.flushPendingStores();
    });

    await server.hooks("onDestroy", { instance: server });
    this.server = null;
  }

  /**
   * Reset the singleton instance (useful for testing)
   */
  public static resetInstance(): void {
    HocusPocusServerManager.instance = null;
  }
}
