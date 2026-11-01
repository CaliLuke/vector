/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { getMetadata } from "reflect-metadata/no-conflict";
import type { RequestHandler } from "express";

function isRequestHandler(value: unknown): value is RequestHandler {
  return typeof value === "function";
}

export function controllerMiddlewares(target: object, propertyKey: string | symbol): RequestHandler[] {
  const value: unknown = getMetadata("middlewares", target, propertyKey);
  if (value === undefined) return [];
  if (!Array.isArray(value) || !value.every(isRequestHandler))
    throw new TypeError("Invalid controller middleware metadata");
  return value;
}
