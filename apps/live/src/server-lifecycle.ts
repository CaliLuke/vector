/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Server } from "./server";

type ManagedServer = Pick<Server, "initialize" | "listen" | "destroy">;

/** Signals share one shutdown, and an initializing server cannot start listening after shutdown begins. */
export class ServerLifecycle {
  private server: ManagedServer | undefined;
  private startup: Promise<void> | undefined;
  private shutdown: Promise<void> | undefined;
  private stopping = false;

  constructor(private readonly createServer: () => ManagedServer) {}

  start() {
    this.startup ??= this.initialize();
    return this.startup;
  }

  stop() {
    this.stopping = true;
    this.shutdown ??= this.destroy();
    return this.shutdown;
  }

  private async initialize() {
    if (this.stopping) return;
    this.server = this.createServer();
    await this.server.initialize();
    if (!this.stopping) this.server.listen();
  }

  private async destroy() {
    // A failed initialization can still have opened resources that need cleanup.
    let startupFailure: { error: unknown } | undefined;
    try {
      await this.startup;
    } catch (error) {
      startupFailure = { error };
    }
    try {
      await this.server?.destroy();
    } catch (error) {
      if (startupFailure)
        throw new AggregateError([startupFailure.error, error], "Startup and cleanup failed", { cause: error });
      throw error;
    }
    if (startupFailure) throw startupFailure.error;
  }
}
