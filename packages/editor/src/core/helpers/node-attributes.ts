/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/** ProseMirror attributes can also come from imported or collaborative JSON. */
export function stringAttribute(attributes: Record<string, unknown>, name: string): string | undefined {
  const value = attributes[name];
  return typeof value === "string" ? value : undefined;
}
