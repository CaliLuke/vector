/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Server as HttpServer } from "node:http";
import type { Router } from "express";
import { createLiveApplication } from "../../src/app";

type LiveTestApplication = ReturnType<typeof createLiveApplication>;

type LiveTestServer = {
  app: LiveTestApplication["app"];
  router: Router;
  baseUrl: string;
  close: () => Promise<void>;
};

export async function startLiveTestServer(
  configure?: (application: LiveTestApplication) => void,
  corsAllowedOrigins = ""
): Promise<LiveTestServer> {
  const application = createLiveApplication({
    port: 0,
    basePath: "/",
    compressionLevel: 6,
    compressionThreshold: 1024,
    corsAllowedOrigins,
  });
  configure?.(application);

  const httpServer: HttpServer = application.app.listen(0);
  await new Promise<void>((resolve, reject) => {
    httpServer.once("listening", resolve);
    httpServer.once("error", reject);
  });

  const address = httpServer.address();
  if (!address || typeof address === "string") throw new Error("Live test server did not bind a TCP port");

  return {
    ...application,
    baseUrl: `http://127.0.0.1:${address.port}`,
    close: async () => {
      if (!httpServer.listening) return;
      httpServer.closeAllConnections();
      await new Promise<void>((resolve, reject) => {
        httpServer.close((error) => (error ? reject(error) : resolve()));
      });
    },
  };
}
