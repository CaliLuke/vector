/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AppError } from "./errors";

export function parsePageDocumentName(name: string): { pageId: string; revision: number } {
  const match = /^([a-zA-Z0-9-]+)(?:\.(0|[1-9][0-9]*))?$/.exec(name);
  const revision = Number(match?.[2] ?? 0);
  if (!match || !Number.isSafeInteger(revision)) {
    throw new AppError("Invalid page document identity", { statusCode: 400 });
  }
  return { pageId: match[1], revision };
}
