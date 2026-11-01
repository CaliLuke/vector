/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RequestHandler, Router, Request } from "express";
import type { WebSocket } from "ws";
import { getMetadata } from "reflect-metadata/no-conflict";
import { controllerMiddlewares } from "./metadata";

export type HttpMethod = "get" | "post" | "put" | "delete" | "patch" | "options" | "head" | "ws";
export type ControllerConstructor<Dependencies extends unknown[] = []> = new (...args: Dependencies) => object;

const restMethods: ReadonlySet<string> = new Set(["get", "post", "put", "delete", "patch", "options", "head"]);
function isRestMethod(method: unknown): method is Exclude<HttpMethod, "ws"> {
  return typeof method === "string" && restMethods.has(method);
}

export function registerController(router: Router, Controller: ControllerConstructor): void;
export function registerController<Dependencies extends unknown[]>(
  router: Router,
  Controller: ControllerConstructor<Dependencies>,
  dependencies: Dependencies
): void;
export function registerController(
  router: Router,
  Controller: new (...args: never[]) => object,
  dependencies: unknown[] = []
): void {
  const instance: object = Reflect.construct(Controller, dependencies);
  const baseRoute: unknown = getMetadata("baseRoute", Controller);
  if (typeof baseRoute !== "string") throw new TypeError(`Missing controller route for ${Controller.name}`);
  const names = Object.getOwnPropertyNames(Controller.prototype).filter((name) => name !== "constructor");
  // Preserve the existing controller-level WebSocket selection contract.
  const websocket = names.some((name) => getMetadata("method", instance, name) === "ws");
  for (const name of names) {
    const method: unknown = getMetadata("method", instance, name);
    const route: unknown = getMetadata("route", instance, name);
    if (typeof route !== "string") continue;
    const handler: unknown = Reflect.get(instance, name);
    if (typeof handler !== "function") continue;
    const path = `${baseRoute}${route}`;
    if (websocket) {
      if (method === "ws" && "ws" in router && typeof router.ws === "function") {
        router.ws(path, (socket: WebSocket, request: Request) => {
          const reportFailure = (error: unknown) => {
            console.error(`WebSocket error in ${Controller.name}.${name}`, error);
            socket.close(1011, "Internal server error");
          };
          try {
            void Promise.resolve(Reflect.apply(handler, instance, [socket, request])).catch(reportFailure);
          } catch (error) {
            reportFailure(error);
          }
        });
      }
    } else if (isRestMethod(method)) {
      const bound: RequestHandler = (request, response, next) =>
        Reflect.apply(handler, instance, [request, response, next]);
      router[method](path, ...controllerMiddlewares(instance, name), bound);
    }
  }
}
