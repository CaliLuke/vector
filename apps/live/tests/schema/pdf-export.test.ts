/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
import { Schema } from "effect";
import { PdfExportRequestBody } from "@/schema/pdf-export";

const decode = Schema.decodeUnknownSync(PdfExportRequestBody);
const validRequest = { pageId: "page-id", workspaceSlug: "workspace" };

describe("PDF export request validation", () => {
  it("accepts the minimal request without inventing optional settings", () => {
    expect(decode(validRequest)).toEqual(validRequest);
  });

  it.each(["A4", "A3", "A2", "LETTER", "LEGAL", "TABLOID"])("accepts supported page size %s", (pageSize) => {
    expect(decode({ ...validRequest, pageSize, pageOrientation: "landscape" })).toEqual({
      ...validRequest,
      pageSize,
      pageOrientation: "landscape",
    });
  });

  it.each([
    { pageId: "" },
    { pageId: " page-id " },
    { workspaceSlug: " " },
    { projectId: "" },
    { pageSize: "B0" },
    { pageOrientation: "diagonal" },
  ])("rejects invalid request fields %j", (fields) => {
    expect(() => decode({ ...validRequest, ...fields })).toThrow();
  });
});
