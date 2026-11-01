import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { selectionText, selectionEntityId, selectionWorkspaceSlug, selectionPriority, selectionWeekStart } =
  await import("../../apps/web/core/components/power-k/core/selection.ts");
const { ShortcutBadge, KeySequenceBadge } =
  await import("../../apps/web/core/components/power-k/ui/modal/command-item-shortcut-badge.tsx");
const { getPowerKSearchSection, isPowerKSearchResultKey } =
  await import("../../apps/web/core/components/power-k/ui/modal/search-results-map.tsx");

void test("command selections reject invalid identifiers and enumerated values before dispatch", () => {
  assert.equal(selectionText("timezone"), "timezone");
  assert.equal(selectionEntityId({ id: "cycle", name: "Cycle" }), "cycle");
  assert.equal(selectionWorkspaceSlug({ slug: "acme", id: "workspace" }), "acme");
  for (const value of [null, undefined, 1, false, {}, { id: 1 }, { id: "" }])
    assert.throws(() => selectionEntityId(value));
  for (const value of [null, undefined, 1, false, {}, { slug: 1 }, { slug: "" }])
    assert.throws(() => selectionWorkspaceSlug(value));
  for (const value of [null, 1, {}, []]) assert.throws(() => selectionText(value));
  for (const value of ["urgent", "high", "medium", "low", "none"]) assert.equal(selectionPriority(value), value);
  for (const value of ["unknown", null, undefined, 1]) assert.throws(() => selectionPriority(value));
  for (const value of [0, 1, 2, 3, 4, 5, 6]) assert.equal(selectionWeekStart(value), value);
  for (const value of ["MONDAY", "1", -1, 7, null]) assert.throws(() => selectionWeekStart(value));
});

void test("shortcut badges preserve whole modifier keys, platform symbols, and repeated sequences", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  Object.defineProperty(navigator, "platform", { configurable: true, value: "Win32" });
  await environment.render(React.createElement(ShortcutBadge, { shortcut: "cmd+shift+," }));
  assert.deepEqual(
    Array.from(environment.container.querySelectorAll("kbd"), (key) => key.textContent),
    ["Ctrl", "Shift", ","]
  );
  Object.defineProperty(navigator, "platform", { configurable: true, value: "MacIntel" });
  await environment.render(React.createElement(ShortcutBadge, { shortcut: "cmd+ctrl+option+shift+escape+arrowup+x" }));
  assert.deepEqual(
    Array.from(environment.container.querySelectorAll("kbd"), (key) => key.textContent),
    ["⌘", "⌃", "⌥", "⇧", "Esc", "↑", "X"]
  );
  await environment.render(React.createElement(KeySequenceBadge, { sequence: "gg" }));
  assert.deepEqual(
    Array.from(environment.container.querySelectorAll("kbd"), (key) => key.textContent),
    ["G", "G"]
  );
  assert.equal(environment.container.textContent, "GthenG");
  await environment.render(React.createElement(ShortcutBadge));
  assert.equal(environment.container.textContent, "");
  await environment.render(React.createElement(KeySequenceBadge));
  assert.equal(environment.container.textContent, "");
});

void test("search sections retain entity paths, matching text, and a page's current project", () => {
  const common = {
    id: "id",
    name: "Name",
    workspace__slug: "acme",
    project_id: "project",
    project__identifier: "PROJ",
  };
  const results = {
    workspace: [{ id: "w", name: "Workspace", slug: "acme" }],
    project: [{ id: "project", name: "Project", workspace__slug: "acme", identifier: "PROJ" }],
    issue: [{ ...common, sequence_id: 7, type_id: "type" }],
    cycle: [common],
    module: [common],
    issue_view: [common],
    page: [
      {
        id: "page",
        name: "Page",
        workspace__slug: "acme",
        project_ids: ["first", "current"],
        project__identifiers: ["PROJ"],
      },
    ],
  };
  const expectedPaths = {
    workspace: "/acme/",
    project: "/acme/projects/project/issues/",
    cycle: "/acme/projects/project/cycles/id",
    module: "/acme/projects/project/modules/id",
    issue_view: "/acme/projects/project/views/id",
    page: "/acme/projects/current/pages/page",
  };
  for (const [key, path] of Object.entries(expectedPaths)) {
    assert.equal(isPowerKSearchResultKey(key), true);
    const section = getPowerKSearchSection(key, results, "current");
    assert.equal(section.key, key);
    assert.equal(section.items[0].path, path);
    assert.ok(section.items[0].label);
  }
  const issue = getPowerKSearchSection("issue", results, undefined).items[0];
  assert.equal(issue.value, "issue-id-Name-PROJ-7");
  assert.ok(issue.path.includes("PROJ-7"));
  assert.equal(getPowerKSearchSection("page", results, "outside").items[0].path, "/acme/projects/first/pages/page");
  results.page[0].project_ids = [];
  assert.equal(getPowerKSearchSection("page", results, undefined).items[0].path, "/acme/wiki/page");
  assert.equal(isPowerKSearchResultKey("unknown"), false);
  assert.equal(isPowerKSearchResultKey("toString"), false);
  results.cycle = [];
  assert.deepEqual(getPowerKSearchSection("cycle", results, undefined).items, []);
});
