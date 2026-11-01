/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { defineMetadata } from "reflect-metadata/no-conflict";
import type { RequestHandler } from "express";
import { controllerMiddlewares } from "./metadata";

// Define valid HTTP methods
type RestMethod = "get" | "post" | "put" | "patch" | "delete";

/**
 * Controller decorator
 * @param baseRoute
 * @returns
 */
export function Controller(baseRoute: string = ""): ClassDecorator {
  return function (target) {
    defineMetadata("baseRoute", baseRoute, target);
  };
}

/**
 * Factory function to create HTTP method decorators
 * @param method HTTP method to handle
 * @returns Method decorator
 */
function createHttpMethodDecorator(method: RestMethod): (route: string) => MethodDecorator {
  return function (route: string): MethodDecorator {
    return function (target: object, propertyKey: string | symbol) {
      defineMetadata("method", method, target, propertyKey);
      defineMetadata("route", route, target, propertyKey);
    };
  };
}

// Export HTTP method decorators using the factory
export const Get = createHttpMethodDecorator("get");
export const Post = createHttpMethodDecorator("post");
export const Put = createHttpMethodDecorator("put");
export const Patch = createHttpMethodDecorator("patch");
export const Delete = createHttpMethodDecorator("delete");

/**
 * Middleware decorator
 * @param middleware
 * @returns
 */
export function Middleware(middleware: RequestHandler): MethodDecorator {
  return function (target: object, propertyKey: string | symbol) {
    defineMetadata("middlewares", [...controllerMiddlewares(target, propertyKey), middleware], target, propertyKey);
  };
}
