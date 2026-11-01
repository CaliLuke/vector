/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { logger } from "@plane/logger";
import { AppError } from "@/lib/errors";
import { Server } from "./server";
import { ServerLifecycle } from "./server-lifecycle";

const lifecycle = new ServerLifecycle(() => new Server());

void lifecycle.start().catch((error: unknown) => {
  logger.error("Failed to start server:", new AppError(error));
  return process.exit(1);
});

let shutdownRequested = false;
function shutdown(signal: "SIGTERM" | "SIGINT") {
  if (shutdownRequested) return;
  shutdownRequested = true;
  logger.info(`Received ${signal} signal. Initiating graceful shutdown...`);
  void lifecycle.stop().then(
    () => {
      logger.info("Server shut down gracefully");
      return process.exit(signal === "SIGTERM" ? 0 : 1);
    },
    (error: unknown) => {
      logger.error("Error during graceful shutdown:", new AppError(error));
      return process.exit(1);
    }
  );
}

process.on("SIGTERM", () => shutdown("SIGTERM"));
process.on("SIGINT", () => shutdown("SIGINT"));

process.on("unhandledRejection", (err: unknown) => {
  const error = new AppError(err);
  logger.error(`[UNHANDLED_REJECTION]`, error);
});

process.on("uncaughtException", (err: unknown) => {
  const error = new AppError(err);
  logger.error(`[UNCAUGHT_EXCEPTION]`, error);
});
