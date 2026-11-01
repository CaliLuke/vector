import { Effect, Result } from "effect";
import { expect, test, vi } from "vitest";
import { parsePdfDocument } from "@/lib/pdf/document";
import { PdfContentFetchError } from "@/schema/pdf-export";
import { ProjectPageService } from "@/services/page/project-page.service";
import { PdfExportService } from "@/services/pdf-export/pdf-export.service";

test("PDF document decoding preserves text, marks, and custom attributes", () => {
  const content = {
    type: "doc",
    content: [
      {
        type: "paragraph",
        attrs: { textAlign: "center" },
        content: [
          {
            type: "text",
            text: "Exported",
            marks: [{ type: "bold" }, { type: "customColor", attrs: { color: "green" } }],
          },
          { type: "imageComponent", attrs: { src: "asset-id", width: 300 } },
        ],
      },
    ],
  };
  expect(parsePdfDocument(content)).toEqual(content);
  expect(parsePdfDocument({ type: "doc" })).toEqual({ type: "doc" });
});

test.each([
  null,
  [],
  { type: "paragraph" },
  { type: "doc", content: {} },
  { type: "doc", content: [{ type: "text", text: 42 }] },
  { type: "doc", content: [{ type: "paragraph", content: [null] }] },
  { type: "doc", content: [{ type: "text", marks: [{ type: 42 }] }] },
])("PDF document decoding rejects malformed content %j", (value) => {
  expect(() => parsePdfDocument(value)).toThrow();
});

test("image collection ignores invalid sources and remote URLs while deduplicating nested assets", async () => {
  const service = await Effect.runPromise(PdfExportService.make);
  const content = parsePdfDocument({
    type: "doc",
    content: [
      ...[42, {}, null, "", "https://example.com/image", "data:image/png;base64,abc", "asset-a"].map((src) => ({
        type: "image",
        attrs: { src },
      })),
      {
        type: "paragraph",
        content: [
          { type: "imageComponent", attrs: { src: "asset-a" } },
          { type: "imageComponent", attrs: { src: "asset-b" } },
        ],
      },
    ],
  });
  expect(service.extractImageAssetIds(content)).toEqual(["asset-a", "asset-b"]);
});

test("invalid stored binary becomes a content error instead of an unhandled export defect", async () => {
  const pageService = new ProjectPageService({
    workspaceSlug: "workspace",
    projectId: "project",
    cookie: "session=test",
  });
  vi.spyOn(pageService, "fetchDescriptionBinary").mockResolvedValue(Buffer.from("invalid Yjs document"));
  try {
    const service = await Effect.runPromise(PdfExportService.make);
    const result = await Effect.runPromise(Effect.result(service.fetchPageContent(pageService, "page", "request")));
    expect(Result.isFailure(result)).toBe(true);
    if (Result.isFailure(result)) expect(result.failure).toBeInstanceOf(PdfContentFetchError);
  } finally {
    vi.restoreAllMocks();
  }
});

test("PDF image URL resolution preserves scope and ordering, then omits unresolved images", async () => {
  const pageService = new ProjectPageService({
    workspaceSlug: "workspace",
    projectId: "project",
    cookie: "session=test",
  });
  const first = deferred<string | null>();
  const started = deferred<void>();
  const resolve = vi.spyOn(pageService, "resolveImageAssetUrl").mockImplementation(async (_workspace, assetId) => {
    if (assetId === "first") {
      started.resolve();
      return first.promise;
    }
    return null;
  });
  try {
    const service = await Effect.runPromise(PdfExportService.make);
    const result = Effect.runPromise(
      service.processImages(pageService, "workspace", "project", ["first", "second"], "request")
    );
    await started.promise;
    expect(resolve).toHaveBeenCalledTimes(1);
    expect(resolve).toHaveBeenNthCalledWith(1, "workspace", "first", "project");
    first.resolve(null);
    expect(await result).toEqual({});
    expect(resolve).toHaveBeenNthCalledWith(2, "workspace", "second", "project");
  } finally {
    vi.restoreAllMocks();
  }
});

test("PDF image URL failures retain the existing empty-image fallback and stop remaining resolutions", async () => {
  const pageService = new ProjectPageService({
    workspaceSlug: "workspace",
    projectId: "project",
    cookie: "session=test",
  });
  const resolve = vi.spyOn(pageService, "resolveImageAssetUrl").mockRejectedValue(new Error("asset unavailable"));
  try {
    const service = await Effect.runPromise(PdfExportService.make);
    expect(
      await Effect.runPromise(
        service.processImages(pageService, "workspace", "project", ["first", "second"], "request")
      )
    ).toEqual({});
    expect(resolve).toHaveBeenCalledTimes(1);
  } finally {
    vi.restoreAllMocks();
  }
});

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((complete) => {
    resolve = complete;
  });
  return { promise, resolve };
}
