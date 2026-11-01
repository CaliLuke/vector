/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/** Keep stored filter order while ignoring fields outside the consumer's contract. */
export function orderedFilterKeys<K extends string>(filters: Partial<Record<K, unknown>>, keys: readonly K[]): K[] {
  return Object.keys(filters).flatMap((key) => {
    const knownKey = keys.find((candidate) => candidate === key);
    return knownKey === undefined ? [] : [knownKey];
  });
}
