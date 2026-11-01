/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { z } from "zod";
import type { TipTapDocument, TipTapNode } from "./types";

const attributes = z.record(z.string(), z.unknown());
const mark = z.object({ type: z.string(), attrs: attributes.optional() });
const node: z.ZodType<TipTapNode> = z.lazy(() =>
  z.object({
    type: z.string(),
    attrs: attributes.optional(),
    content: z.array(node).optional(),
    text: z.string().optional(),
    marks: z.array(mark).optional(),
  })
);
const document = z.object({ type: z.literal("doc"), content: z.array(node).optional() });

/** Validate decoded storage content before the PDF renderer traverses it. */
export function parsePdfDocument(value: unknown): TipTapDocument {
  return document.parse(value);
}
