/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export function toggleCatalogueFilterValues(current: readonly string[] | null | undefined, value: string | string[]) {
  const next = [...(current ?? [])];
  const values = Array.isArray(value) ? value : [value];

  for (const selected of values) {
    const index = next.indexOf(selected);
    if (index === -1) next.push(selected);
    else next.splice(index, 1);
  }

  return next;
}
