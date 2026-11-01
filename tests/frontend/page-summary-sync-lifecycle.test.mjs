import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { PageSyncingBadge } = await import("../../apps/web/core/components/pages/header/syncing-badge.tsx");
const { PageContentBrowser } = await import("../../apps/web/core/components/pages/editor/summary/content-browser.tsx");

function editorFixture(text) {
  let headings = [{ level: 1, sequence: 1, text }];
  const listeners = new Set();
  const scrolled = [];
  return {
    listeners,
    scrolled,
    getHeadings: () => headings,
    onHeadingChange(callback) {
      listeners.add(callback);
      return () => listeners.delete(callback);
    },
    scrollSummary: (heading) => scrolled.push(heading),
    update(nextText) {
      headings = [{ level: 2, sequence: 1, text: nextText }];
      listeners.forEach((listener) => listener(headings));
    },
  };
}

void test("a completed sync cannot hide a subsequent syncing or error badge", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const renderStatus = (syncStatus) => environment.render(React.createElement(PageSyncingBadge, { syncStatus }));
  await renderStatus("syncing");
  assert.match(environment.container.textContent, /Syncing/);
  await renderStatus("synced");
  assert.equal(environment.container.textContent, "");
  await renderStatus("error");
  await React.act(async () => t.mock.timers.tick(300));
  assert.match(environment.container.textContent, /Connection lost/);
  await renderStatus("syncing");
  assert.match(environment.container.textContent, /Syncing/);
});

void test("page outlines follow their editor subscription and release replaced editors", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const first = editorFixture("First heading");
  const second = editorFixture("Replacement heading");
  const peeks = [];
  const renderEditor = (editorRef) =>
    environment.render(
      React.createElement(PageContentBrowser, {
        editorRef,
        emptyState: "No headings",
        setSidePeekVisible: (value) => peeks.push(value),
      })
    );
  await renderEditor(first);
  assert.equal(environment.container.textContent, "First heading");
  await React.act(async () => first.update("Edited heading"));
  assert.equal(environment.container.textContent, "Edited heading");
  await React.act(async () => environment.container.querySelector("button").click());
  assert.equal(first.scrolled[0].text, "Edited heading");
  assert.deepEqual(peeks, [false]);
  await renderEditor(second);
  assert.equal(first.listeners.size, 0);
  assert.equal(environment.container.textContent, "Replacement heading");
  await renderEditor(null);
  assert.equal(second.listeners.size, 0);
  assert.equal(environment.container.textContent, "No headings");
});
