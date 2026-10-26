/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export function getModulePeekHref(pathname: string, search: string, moduleId?: string) {
  const params = new URLSearchParams(search);
  if (moduleId && params.get("peekModule") !== moduleId) params.set("peekModule", moduleId);
  else params.delete("peekModule");

  const query = params.toString();
  return query ? `${pathname}?${query}` : pathname;
}
