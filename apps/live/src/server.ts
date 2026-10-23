/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Server as HttpServer } from "http";
import type { Hocuspocus } from "@hocuspocus/server";
import type { Express, Request, Response, Router } from "express";
// plane imports
import { registerController } from "@plane/decorators";
import { logger } from "@plane/logger";
import { createLiveApplication } from "@/app";
// controllers
import { CONTROLLERS } from "@/controllers";
// env
import { env } from "@/env";
// hocuspocus server
import { HocusPocusServerManager } from "@/hocuspocus";
// redis
import { redisManager } from "@/redis";

export class Server {
  private app: Express;
  private router: Router;
  private hocuspocusServer: Hocuspocus | undefined;
  private httpServer: HttpServer | undefined;

  constructor() {
    const application = createLiveApplication({
      port: env.PORT || 3000,
      basePath: env.LIVE_BASE_PATH,
      compressionLevel: env.COMPRESSION_LEVEL,
      compressionThreshold: env.COMPRESSION_THRESHOLD,
      corsAllowedOrigins: env.CORS_ALLOWED_ORIGINS,
    });
    this.app = application.app;
    this.router = application.router;
  }

  public async initialize(): Promise<void> {
    try {
      await redisManager.initialize();
      logger.info("SERVER: Redis setup completed");
      const manager = HocusPocusServerManager.getInstance();
      this.hocuspocusServer = await manager.initialize();
      logger.info("SERVER: HocusPocus setup completed");
      this.setupRoutes(this.hocuspocusServer);
      this.setupNotFoundHandler();
    } catch (error) {
      logger.error("SERVER: Failed to initialize live server dependencies:", error);
      throw error;
    }
  }

  private setupNotFoundHandler() {
    this.app.use((_req: Request, res: Response) => {
      res.status(404).json({
        message: "Not Found",
      });
    });
  }

  private setupRoutes(hocuspocusServer: Hocuspocus) {
    CONTROLLERS.forEach((controller) => registerController(this.router, controller, [hocuspocusServer]));
  }

  public listen() {
    this.httpServer = this.app.listen(this.app.get("port"), (error?: Error) => {
      if (error) {
        logger.error("SERVER: Failed to start server:", error);
        throw error;
      }
      logger.info(`SERVER: Express server has started at port ${this.app.get("port")}`);
    });
  }

  public async destroy() {
    if (this.hocuspocusServer) {
      await HocusPocusServerManager.getInstance().destroy();
      logger.info("SERVER: HocusPocus connections closed gracefully.");
    }

    await redisManager.disconnect();
    logger.info("SERVER: Redis connection closed gracefully.");

    if (this.httpServer) {
      await new Promise<void>((resolve, reject) => {
        this.httpServer!.close((err) => {
          if (err) {
            reject(err);
          } else {
            logger.info("SERVER: Express server closed gracefully.");
            resolve();
          }
        });
      });
    }
  }
}
