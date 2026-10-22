import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as Y from "yjs";
import type { Document, Hocuspocus } from "@hocuspocus/server";
import { Database } from "@/extensions/database";
import { TitleSyncExtension } from "@/extensions/title-sync";
import { TitleUpdateManager } from "@/extensions/title-update/title-update-manager";
import { getPageService } from "@/services/page/handler";
import type { HocusPocusServerContext } from "@/types";

vi.mock("@/services/page/handler", () => ({
  getPageService: vi.fn(),
}));

vi.mock("@/utils/broadcast-message", () => ({
  broadcastMessageToPage: vi.fn(),
}));

vi.mock("@/extensions/force-close-handler", () => ({
  forceCloseDocumentAcrossServers: vi.fn(),
}));

const pageId = "page-id";
const context: HocusPocusServerContext = {
  projectId: "project-id",
  cookie: "session-cookie",
  documentType: "project_page",
  workspaceSlug: "workspace",
  userId: "user-id",
};

const makeService = () => ({
  fetchDescriptionBinary: vi.fn(async () => Buffer.alloc(0)),
  fetchDetails: vi.fn(async () => ({
    id: pageId,
    name: "Loaded title",
    description_html: "<p>Readable locked-page content</p>",
  })),
  updateDescriptionBinary: vi.fn(async () => undefined),
  updatePageProperties: vi.fn(async (): Promise<void> => undefined),
});

const titleFragment = (document: Y.Doc) => document.getXmlFragment("title");

const replaceTitle = (document: Y.Doc, title: string) => {
  const fragment = titleFragment(document);
  if (fragment.length > 0) fragment.delete(0, fragment.length);
  const text = new Y.XmlText();
  text.insert(0, title);
  fragment.insert(0, [text]);
};

describe("page persistence during document load and title sync", () => {
  let service: ReturnType<typeof makeService>;

  beforeEach(() => {
    service = makeService();
    vi.mocked(getPageService).mockReturnValue(service as unknown as ReturnType<typeof getPageService>);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("hydrates converted HTML on a read and persists it only on a later document edit", async () => {
    const database = new Database();
    const fetchPayload = {
      context,
      documentName: pageId,
      instance: {} as Hocuspocus,
    } as Parameters<typeof database.configuration.fetch>[0];

    const loadedState = await database.configuration.fetch(fetchPayload);

    expect(loadedState).toBeInstanceOf(Uint8Array);
    expect(service.updateDescriptionBinary).not.toHaveBeenCalled();

    const document = new Y.Doc();
    Y.applyUpdate(document, loadedState as Uint8Array);
    expect(document.getXmlFragment("default").length).toBeGreaterThan(0);

    document.getMap("user-edits").set("edited", true);
    await database.onStoreDocument({
      clientsCount: 1,
      context,
      document: document as unknown as Document,
      documentName: pageId,
      instance: {} as Hocuspocus,
      requestHeaders: {},
      requestParameters: new URLSearchParams(),
      socketId: "socket-id",
    } as Parameters<typeof database.onStoreDocument>[0]);

    expect(service.updateDescriptionBinary).toHaveBeenCalledOnce();
  });

  it("ignores an observed title equal to the loaded Yjs title", async () => {
    const document = new Y.Doc();
    replaceTitle(document, "Loaded title");
    const extension = new TitleSyncExtension();
    await extension.afterLoadDocument({
      document: document as unknown as Document,
      documentName: pageId,
      context,
      instance: {} as Hocuspocus,
    });

    replaceTitle(document, "Loaded title");
    await extension.beforeUnloadDocument({ documentName: pageId });

    expect(service.updatePageProperties).not.toHaveBeenCalled();
    await extension.afterUnloadDocument({ documentName: pageId, document: document as unknown as Document });
  });

  it("persists a changed title after the debounce interval", async () => {
    const document = new Y.Doc();
    replaceTitle(document, "Loaded title");
    const extension = new TitleSyncExtension();
    await extension.afterLoadDocument({
      document: document as unknown as Document,
      documentName: pageId,
      context,
      instance: {} as Hocuspocus,
    });

    vi.useFakeTimers();
    replaceTitle(document, "Edited title");
    await vi.advanceTimersByTimeAsync(5000);

    expect(service.updatePageProperties).toHaveBeenCalledOnce();
    expect(service.updatePageProperties).toHaveBeenCalledWith(pageId, {
      data: { name: "Edited title" },
      abortSignal: expect.any(AbortSignal),
    });
    await extension.beforeUnloadDocument({ documentName: pageId });
    await extension.afterUnloadDocument({ documentName: pageId, document: document as unknown as Document });
    vi.useRealTimers();
  });

  it("flushes a pending changed title before the document unloads", async () => {
    const document = new Y.Doc();
    replaceTitle(document, "Loaded title");
    const extension = new TitleSyncExtension();
    await extension.afterLoadDocument({
      document: document as unknown as Document,
      documentName: pageId,
      context,
      instance: {} as Hocuspocus,
    });

    replaceTitle(document, "Unloaded title");
    await extension.beforeUnloadDocument({ documentName: pageId });

    expect(service.updatePageProperties).toHaveBeenCalledOnce();
    expect(service.updatePageProperties).toHaveBeenCalledWith(
      pageId,
      expect.objectContaining({ data: { name: "Unloaded title" } })
    );
    await extension.afterUnloadDocument({ documentName: pageId, document: document as unknown as Document });
  });

  it("cancels a pending title when it returns to the loaded value", async () => {
    const document = new Y.Doc();
    replaceTitle(document, "Loaded title");
    const extension = new TitleSyncExtension();
    await extension.afterLoadDocument({
      document: document as unknown as Document,
      documentName: pageId,
      context,
      instance: {} as Hocuspocus,
    });

    replaceTitle(document, "Temporary title");
    replaceTitle(document, "Loaded title");
    replaceTitle(document, "Final title");
    await extension.beforeUnloadDocument({ documentName: pageId });

    expect(service.updatePageProperties).toHaveBeenCalledOnce();
    expect(service.updatePageProperties).toHaveBeenCalledWith(
      pageId,
      expect.objectContaining({ data: { name: "Final title" } })
    );
    await extension.afterUnloadDocument({ documentName: pageId, document: document as unknown as Document });
  });

  it("retries a failed title save because the loaded title remains the baseline", async () => {
    service.updatePageProperties
      .mockRejectedValueOnce(new Error("temporary API failure"))
      .mockResolvedValueOnce(undefined);
    const manager = new TitleUpdateManager(pageId, context, 60_000, "Loaded title");
    manager.scheduleUpdate("Edited title");
    await manager.forceSave();
    manager.scheduleUpdate("Edited title");
    await manager.forceSave();

    expect(service.updatePageProperties).toHaveBeenCalledTimes(2);
    manager.cancel();
  });

  it("writes a title revert after an earlier title save already started", async () => {
    vi.useFakeTimers();
    let finishFirstSave: (() => void) | undefined;
    service.updatePageProperties
      .mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            finishFirstSave = resolve;
          })
      )
      .mockResolvedValueOnce(undefined);
    const manager = new TitleUpdateManager(pageId, context, 5000, "Loaded title");
    manager.scheduleUpdate("In-flight title");
    await vi.advanceTimersByTimeAsync(5000);
    expect(service.updatePageProperties).toHaveBeenCalledOnce();

    manager.scheduleUpdate("Loaded title");
    finishFirstSave?.();
    await vi.advanceTimersByTimeAsync(0);
    await vi.advanceTimersByTimeAsync(5000);

    expect(service.updatePageProperties).toHaveBeenCalledTimes(2);
    expect(service.updatePageProperties).toHaveBeenLastCalledWith(
      pageId,
      expect.objectContaining({ data: { name: "Loaded title" } })
    );
    manager.cancel();
    vi.useRealTimers();
  });

  it("force-saves an empty title instead of treating it as no pending update", async () => {
    const manager = new TitleUpdateManager(pageId, context, 60_000, "Loaded title");
    manager.scheduleUpdate("");
    await manager.forceSave();

    expect(service.updatePageProperties).toHaveBeenCalledWith(pageId, {
      data: { name: "" },
      abortSignal: expect.any(AbortSignal),
    });
    manager.cancel();
  });
});
