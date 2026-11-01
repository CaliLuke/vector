/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Handle } from "hast-util-to-mdast";

// Markdown has no underline; retain the children and let the standard mark handlers preserve nesting.
export const parseMarks: Record<string, Handle> = {
  u: (state, node) => state.all(node),
};
