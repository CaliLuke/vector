/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { registerController } from "@plane/decorators";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { getBinaryDataFromDocumentEditorHTMLString } from "@plane/editor/lib";
import { PdfExportController } from "../../src/controllers/pdf-export.controller";
import { renderPlaneDocToPdfBuffer } from "../../src/lib/pdf";
import { getPageService } from "../../src/services/page/handler";
import { ProjectPageService } from "../../src/services/page/project-page.service";
import { startLiveTestServer } from "../helpers/http-server";

vi.mock("../../src/lib/pdf", () => ({
  renderPlaneDocToPdfBuffer: vi.fn(),
}));
vi.mock("../../src/services/page/handler", () => ({
  getPageService: vi.fn(),
}));

const pdfBuffer = Buffer.from("%PDF-fake-document");
const storedPageBinary = Buffer.from(
  getBinaryDataFromDocumentEditorHTMLString("<p>Stored page body</p>", "Stored title")
);

describe("PDF export HTTP contract", () => {
  let closeServer: () => Promise<void>;
  let baseUrl: string;

  beforeAll(async () => {
    const server = await startLiveTestServer(({ router }) => {
      registerController(router, PdfExportController);
    });
    closeServer = server.close;
    baseUrl = server.baseUrl;
  });

  afterAll(async () => {
    await closeServer?.();
  });

  beforeEach(() => {
    vi.mocked(getPageService).mockReset();
    vi.mocked(renderPlaneDocToPdfBuffer).mockReset();

    const pageService = new ProjectPageService({
      workspaceSlug: "workspace",
      projectId: "project-456",
      cookie: "session=test",
    });
    vi.spyOn(pageService, "fetchDescriptionBinary").mockResolvedValue(storedPageBinary);
    vi.spyOn(pageService, "fetchUserMentions").mockResolvedValue([]);
    vi.mocked(getPageService).mockReturnValue(pageService);
    vi.mocked(renderPlaneDocToPdfBuffer).mockResolvedValue(pdfBuffer);
  });

  it("rejects requests without a session cookie before validating the body", async () => {
    const response = await fetch(`${baseUrl}/pdf-export`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    });

    expect(response.status).toBe(401);
    expect(await response.json()).toEqual({ message: "Authentication required" });
    expect(getPageService).not.toHaveBeenCalled();
  });

  it("rejects malformed request fields without fetching page content", async () => {
    const response = await fetch(`${baseUrl}/pdf-export`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Cookie: "session=test" },
      body: JSON.stringify({ pageId: " ", workspaceSlug: "workspace" }),
    });

    expect(response.status).toBe(400);
    expect(await response.json()).toEqual({ message: "Invalid request body" });
    expect(getPageService).not.toHaveBeenCalled();
  });

  it("returns the rendered PDF and safely encodes a Unicode attachment filename", async () => {
    const response = await fetch(`${baseUrl}/pdf-export`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Cookie: "session=test" },
      body: JSON.stringify({
        pageId: "page-123",
        workspaceSlug: "workspace",
        projectId: "project-456",
        title: "Export title",
        author: "Test author",
        subject: "Test subject",
        pageSize: "A3",
        pageOrientation: "landscape",
        fileName: 'Rêport "ä".pdf',
      }),
    });

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBe("application/pdf");
    expect(response.headers.get("content-length")).toBe(String(pdfBuffer.length));
    expect(response.headers.get("content-disposition")).toBe(
      "attachment; filename=\"R_port _.pdf\"; filename*=UTF-8''R%C3%AAport%20%22%C3%A4%22.pdf"
    );
    expect(Buffer.from(await response.arrayBuffer())).toEqual(pdfBuffer);

    expect(getPageService).toHaveBeenCalledWith("project_page", {
      workspaceSlug: "workspace",
      projectId: "project-456",
      cookie: "session=test",
      documentType: "project_page",
      userId: "",
    });
    const [document, options] = vi.mocked(renderPlaneDocToPdfBuffer).mock.calls[0] ?? [];
    expect(document).toMatchObject({
      type: "doc",
      content: [
        {
          type: "paragraph",
          content: [{ type: "text", text: "Stored page body" }],
        },
      ],
    });
    expect(options).toEqual(
      expect.objectContaining({
        title: "Export title",
        author: "Test author",
        subject: "Test subject",
        pageSize: "A3",
        pageOrientation: "landscape",
      })
    );
  });
});
