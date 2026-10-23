/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import compression from "compression";
import cors from "cors";
import express from "express";
import type { Express } from "express";
import expressWs from "express-ws";
import type { Router } from "express-ws";
import helmet from "helmet";
import { loggerMiddleware } from "@plane/logger";

type LiveApplicationConfig = {
  port: number | string;
  basePath: string;
  compressionLevel: number;
  compressionThreshold: number;
  corsAllowedOrigins: string;
};

type LiveApplication = {
  app: Express;
  router: Router;
};

export function createLiveApplication(config: LiveApplicationConfig): LiveApplication {
  const app = express();
  // Explicit router setup avoids the Express 5 prototype mismatch:
  // https://github.com/HenningM/express-ws/issues/169
  // Revisit when the default setup supports Express 5 router instances.
  const webSockets = expressWs(app, undefined, { leaveRouterUntouched: true });

  app.use(helmet());
  app.use(compression({ level: config.compressionLevel, threshold: config.compressionThreshold }));
  app.use(loggerMiddleware);
  app.use(express.json());
  app.use(express.urlencoded({ extended: true }));

  const allowedOrigins = config.corsAllowedOrigins.split(",").map((origin) => origin.trim());
  app.use(
    cors({
      origin: allowedOrigins,
      credentials: true,
      methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
      allowedHeaders: ["Content-Type", "Authorization", "x-api-key"],
    })
  );

  const router = express.Router();
  webSockets.applyTo(router);
  app.set("port", config.port);
  app.use(config.basePath, router);

  return { app, router };
}
