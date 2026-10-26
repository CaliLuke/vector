/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
export function validateTitle(value: string) {
  if (!value.trim()) return "title_is_required";
  if (value.length > 255) return "title_should_be_less_than_255_characters";
  return undefined;
}
