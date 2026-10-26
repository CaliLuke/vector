import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
globalThis.pageStatusControls = { editor: null, copyLink: async () => true };
const stubs = {
  "@/hooks/use-page-permissions": "export const usePagePermissions=()=>({canCurrentUserLockPage:true});",
  "@/hooks/use-page-operations": `export const usePageOperations=()=>({pageOperations:{
    toggleLock:async()=>{}, copyLink:()=>globalThis.pageStatusControls.copyLink()
  }});`,
  "@/components/pages/page-record-context":
    "export const usePageRecord=()=>({editorRef:globalThis.pageStatusControls.editor});",
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const page = { id: "page-1", is_locked: false };

await test("page copy feedback waits for clipboard success and stays clear on failure", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { PageCopyLinkControl } = await import("../../apps/web/core/components/pages/header/copy-link-control.tsx");
  let pending = Promise.withResolvers();
  globalThis.pageStatusControls.copyLink = () => pending.promise;
  await environment.render(React.createElement(PageCopyLinkControl, { page, workspaceSlug: "acme" }));
  const button = environment.container.querySelector("button");
  await act(async () => button.click());
  assert.equal(button.getAttribute("aria-label"), "Copy link");
  await act(async () => pending.resolve(false));
  assert.equal(button.getAttribute("aria-label"), "Copy link");
  pending = Promise.withResolvers();
  await act(async () => button.click());
  await act(async () => pending.resolve(true));
  assert.equal(button.getAttribute("aria-label"), "Copied link");
});

await test("page lock feedback follows canonical transitions and resets for a different page", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { PageLockControl } = await import("../../apps/web/core/components/pages/header/lock-control.tsx");
  const render = (record) =>
    environment.render(React.createElement(PageLockControl, { page: record, workspaceSlug: "acme" }));
  await render({ ...page, is_locked: true });
  assert.ok(environment.container.querySelector('[aria-label="Locked"]'));
  t.mock.timers.enable({ apis: ["setTimeout"] });
  await render(page);
  assert.ok(environment.container.querySelector('[aria-label="Unlocked"]'));
  await act(async () => t.mock.timers.tick(600));
  assert.ok(environment.container.querySelector('button[aria-label="Lock"]'));
  await render({ ...page, is_locked: true });
  await render({ ...page, id: "page-2" });
  assert.ok(environment.container.querySelector('button[aria-label="Lock"]'));
  assert.equal(environment.container.querySelector('[aria-label="Unlocked"]'), null);
});

function editorFixture(words) {
  let info = { words, characters: words * 5, paragraphs: 1 };
  const listeners = new Set();
  return {
    getDocumentInfo: () => ({ ...info }),
    onDocumentInfoChange: (callback) => {
      listeners.add(callback);
      return () => listeners.delete(callback);
    },
    update: (wordCount) => {
      info = { ...info, words: wordCount };
      for (const callback of listeners) callback({ ...info });
    },
    listenerCount: () => listeners.size,
  };
}

await test("page document information follows the active editor and releases old subscriptions", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { PageNavigationPaneInfoTabDocumentInfo } =
    await import("../../apps/web/core/components/pages/navigation-pane/tab-panels/info/document-info.tsx");
  const first = editorFixture(12);
  const second = editorFixture(42);
  globalThis.pageStatusControls.editor = first;
  const render = () => environment.render(React.createElement(PageNavigationPaneInfoTabDocumentInfo));
  const words = () => environment.container.querySelector("h6").textContent;
  await render();
  assert.equal(words(), "12");
  await act(async () => first.update(20));
  assert.equal(words(), "20");
  globalThis.pageStatusControls.editor = second;
  await render();
  assert.equal(first.listenerCount(), 0);
  assert.equal(second.listenerCount(), 1);
  assert.equal(words(), "42");
  await act(async () => first.update(99));
  assert.equal(words(), "42");
  await environment.unmount();
  assert.equal(second.listenerCount(), 0);
});
