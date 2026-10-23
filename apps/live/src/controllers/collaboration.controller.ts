/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Hocuspocus } from "@hocuspocus/server";
import type { Request } from "express";
import type WebSocket from "ws";
// plane imports
import { Controller, WebSocket as WSDecorator } from "@plane/decorators";
import { logger } from "@plane/logger";

type ExpressRequestData = Pick<Request, "headers" | "method" | "originalUrl" | "protocol"> & {
  get(name: string): string | undefined;
};

export const toHocuspocusRequest = (req: ExpressRequestData): globalThis.Request => {
  const requestHeaders = new Headers();
  for (const [name, value] of Object.entries(req.headers)) {
    if (value !== undefined) requestHeaders.set(name, Array.isArray(value) ? value.join(", ") : value);
  }

  const requestUrl = new URL(req.originalUrl, `${req.protocol}://${req.get("host") || "localhost"}`);
  return new globalThis.Request(requestUrl, { headers: requestHeaders, method: req.method });
};

@Controller("/collaboration")
export class CollaborationController {
  [key: string]: unknown;
  private readonly hocusPocusServer: Hocuspocus;

  constructor(hocusPocusServer: Hocuspocus) {
    this.hocusPocusServer = hocusPocusServer;
  }

  @WSDecorator("/")
  handleConnection(ws: WebSocket, req: Request) {
    try {
      // Hocuspocus v4 consumes the web-standard Request API.
      const connection = this.hocusPocusServer.handleConnection(ws, toHocuspocusRequest(req));
      // Custom transports own event delivery and cleanup in Hocuspocus v4.
      ws.on("message", (data) => {
        const message = Array.isArray(data) ? Buffer.concat(data) : new Uint8Array(data);
        connection.handleMessage(message);
      });
      ws.on("pong", () => connection.touch());
      ws.once("close", (code, reason) => connection.handleClose({ code, reason: reason.toString() }));

      // Set up error handling for the connection
      ws.on("error", (error: Error) => {
        logger.error("COLLABORATION_CONTROLLER: WebSocket connection error:", error);
        ws.close(1011, "Internal server error");
      });
    } catch (error) {
      logger.error("COLLABORATION_CONTROLLER: WebSocket connection error:", error);
      ws.close(1011, "Internal server error");
    }
  }
}
