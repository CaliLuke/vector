import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as Y from "yjs";
import { Document, Hocuspocus } from "@hocuspocus/server";
import type { TPage } from "@plane/types";
import { Database } from "@/extensions/database";
import { TitleSyncExtension } from "@/extensions/title-sync";
import { TitleUpdateManager } from "@/extensions/title-update/title-update-manager";
import { forceCloseDocumentAcrossServers } from "@/extensions/force-close-handler";
import { AppError } from "@/lib/errors";
import { getPageService } from "@/services/page/handler";
import { ProjectPageService } from "@/services/page/project-page.service";
import type { HocusPocusServerContext } from "@/types";
import { CloseCode, ForceCloseReason } from "@/types/admin-commands";

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
const createServer = () => new Hocuspocus();
const context: HocusPocusServerContext = {
  projectId: "project-id",
  cookie: "session-cookie",
  documentType: "project_page",
  workspaceSlug: "workspace",
  userId: "user-id",
};

const makePage = (): TPage => ({
  id: pageId,
  name: "Loaded title",
  description_html: "<p>Readable locked-page content</p>",
  access: undefined,
  archived_at: undefined,
  color: undefined,
  created_at: undefined,
  created_by: undefined,
  description_json: undefined,
  is_favorite: false,
  is_locked: false,
  label_ids: undefined,
  owned_by: undefined,
  updated_at: undefined,
  updated_by: undefined,
  workspace: undefined,
  logo_props: undefined,
  deleted_at: undefined,
});

const makeService = () => {
  const instance = new ProjectPageService({
    workspaceSlug: "workspace",
    projectId: "project-id",
    cookie: "session-cookie",
  });
  const fetchDescriptionBinary = vi.fn(async () => Buffer.alloc(0));
  const fetchDetails = vi.fn(async () => makePage());
  const updateDescriptionBinary = vi.fn(async () => undefined);
  const updatePageProperties = vi.fn(async () => makePage());
  Object.assign(instance, { fetchDescriptionBinary, fetchDetails, updateDescriptionBinary, updatePageProperties });
  return { instance, fetchDescriptionBinary, fetchDetails, updateDescriptionBinary, updatePageProperties };
};

const titleFragment = (document: Y.Doc) => document.getXmlFragment("title");

const replaceTitle = (document: Y.Doc, title: string) => {
  const fragment = titleFragment(document);
  if (fragment.length > 0) fragment.delete(0, fragment.length);
  const text = new Y.XmlText();
  text.insert(0, title);
  fragment.insert(0, [text]);
};

const expectTitleSaved = (service: ReturnType<typeof makeService>, title: string) => {
  expect(service.updatePageProperties).toHaveBeenCalledOnce();
  expect(service.updatePageProperties).toHaveBeenCalledWith(pageId, expect.objectContaining({ data: { name: title } }));
};

const createLoadedDocument = async () => {
  const document = new Document(pageId);
  replaceTitle(document, "Loaded title");
  const extension = new TitleSyncExtension();
  await extension.afterLoadDocument({
    document,
    documentName: pageId,
    context,
    instance: createServer(),
  });
  return { document, extension };
};

describe("page persistence during document load and title sync", () => {
  let service: ReturnType<typeof makeService>;

  beforeEach(() => {
    service = makeService();
    vi.mocked(getPageService).mockReturnValue(service.instance);
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
      instance: createServer(),
      document: new Document(pageId),
      requestHeaders: new Headers(),
      requestParameters: new URLSearchParams(),
      socketId: "socket-id",
      connectionConfig: { isAuthenticated: true, readOnly: false },
    };

    const loadedState = await database.configuration.fetch(fetchPayload);

    if (!(loadedState instanceof Uint8Array)) throw new Error("Document fetch did not return a Yjs update");
    expect(service.updateDescriptionBinary).not.toHaveBeenCalled();

    const document = new Document(pageId);
    Y.applyUpdate(document, loadedState);
    expect(document.getXmlFragment("default").length).toBeGreaterThan(0);

    document.getMap("user-edits").set("edited", true);
    vi.mocked(getPageService).mockClear();
    await database.onStoreDocument({
      clientsCount: 1,
      lastContext: context,
      document,
      documentName: pageId,
      instance: createServer(),
      lastTransactionOrigin: null,
    });

    expect(getPageService).toHaveBeenCalledWith("project_page", context);
    expect(service.updateDescriptionBinary).toHaveBeenCalledOnce();
  });

  it("disconnects and unloads a replaced document when its pending body save is rejected", async () => {
    service.updateDescriptionBinary.mockRejectedValueOnce(new AppError("Page replaced", { statusCode: 409 }));
    const database = new Database();
    const instance = createServer();
    const document = new Document(pageId);
    await expect(
      database.onStoreDocument({
        clientsCount: 1,
        lastContext: context,
        document,
        documentName: pageId,
        instance,
        lastTransactionOrigin: null,
      })
    ).resolves.toBeUndefined();
    expect(service.updateDescriptionBinary).toHaveBeenCalledOnce();
    expect(forceCloseDocumentAcrossServers).toHaveBeenCalledExactlyOnceWith(
      instance,
      pageId,
      ForceCloseReason.CRITICAL_ERROR,
      CloseCode.FORCE_CLOSE
    );
  });

  it("ignores an observed title equal to the loaded Yjs title", async () => {
    const { document, extension } = await createLoadedDocument();

    replaceTitle(document, "Loaded title");
    await extension.beforeUnloadDocument({ documentName: pageId });

    expect(service.updatePageProperties).not.toHaveBeenCalled();
    await extension.afterUnloadDocument({ documentName: pageId, document });
  });

  it("persists a changed title after the debounce interval", async () => {
    const { document, extension } = await createLoadedDocument();

    vi.useFakeTimers();
    replaceTitle(document, "Edited title");
    await vi.advanceTimersByTimeAsync(5000);

    expect(service.updatePageProperties).toHaveBeenCalledOnce();
    expect(service.updatePageProperties).toHaveBeenCalledWith(pageId, {
      data: { name: "Edited title" },
      abortSignal: expect.any(AbortSignal),
    });
    await extension.beforeUnloadDocument({ documentName: pageId });
    await extension.afterUnloadDocument({ documentName: pageId, document });
    vi.useRealTimers();
  });

  it("flushes a pending changed title before the document unloads", async () => {
    const { document, extension } = await createLoadedDocument();

    replaceTitle(document, "Unloaded title");
    await extension.beforeUnloadDocument({ documentName: pageId });

    expectTitleSaved(service, "Unloaded title");
    await extension.afterUnloadDocument({ documentName: pageId, document });
  });

  it("cancels a pending title when it returns to the loaded value", async () => {
    const { document, extension } = await createLoadedDocument();

    replaceTitle(document, "Temporary title");
    replaceTitle(document, "Loaded title");
    replaceTitle(document, "Final title");
    await extension.beforeUnloadDocument({ documentName: pageId });

    expectTitleSaved(service, "Final title");
    await extension.afterUnloadDocument({ documentName: pageId, document });
  });

  it("retries a failed title save because the loaded title remains the baseline", async () => {
    service.updatePageProperties
      .mockRejectedValueOnce(new Error("temporary API failure"))
      .mockResolvedValueOnce(makePage());
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
    let finishFirstSave: ((value: TPage) => void) | undefined;
    service.updatePageProperties
      .mockImplementationOnce(
        () =>
          new Promise<TPage>((resolve) => {
            finishFirstSave = resolve;
          })
      )
      .mockResolvedValueOnce(makePage());
    const manager = new TitleUpdateManager(pageId, context, 5000, "Loaded title");
    manager.scheduleUpdate("In-flight title");
    await vi.advanceTimersByTimeAsync(5000);
    expect(service.updatePageProperties).toHaveBeenCalledOnce();

    manager.scheduleUpdate("Loaded title");
    finishFirstSave?.(makePage());
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
