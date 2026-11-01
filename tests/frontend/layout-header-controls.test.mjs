import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
globalThis.layoutHeaderFixture = { route: { workspaceSlug: "workspace", projectId: "project" }, modals: {} };
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@/hooks/use-issue-list-type": "export const useIssueListType=()=> 'project';",
  "@/components/issues/issue-modal/modal":
    "export const CreateUpdateIssueModal=props=>{globalThis.layoutHeaderFixture.modals.issue=props;return null;};",
  "@/components/epic-modal":
    "export const CreateUpdateEpicModal=props=>{globalThis.layoutHeaderFixture.modals.epic=props;return null;};",
  "@/components/core/modals/existing-issues-list-modal":
    "export const ExistingIssuesListModal=props=>{globalThis.layoutHeaderFixture.modals.existing=props;return null;};",
  "@/components/core/multiple-select": "export const MultipleSelectGroupAction=()=>null;",
});
registerHooks({
  resolve(specifier, context, next) {
    if (
      specifier === "@tanstack/react-router" &&
      (context.parentURL?.endsWith("/headers/group-by-card.tsx") ||
        context.parentURL?.endsWith("/issue-layouts/group-work-item-modals.tsx"))
    ) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const useParams=()=>globalThis.layoutHeaderFixture.route;")}`,
      };
    }
    return next(specifier, context);
  },
});
const bootstrap = await createReactDomTestEnvironment();
const { HeaderColumn } =
  await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/columns/header-column.tsx");
const { HeaderGroupByCard } =
  await import("../../apps/web/core/components/issues/issue-layouts/kanban/headers/group-by-card.tsx");
const { HeaderGroupByCard: ListGroupByCard } =
  await import("../../apps/web/core/components/issues/issue-layouts/list/headers/group-by-card.tsx");
const { SPREADSHEET_PROPERTY_DETAILS } = await import("../../packages/constants/dist/index.mjs");
await bootstrap.unmount();
async function openMenu(env) {
  const trigger = env.container.querySelector('button[aria-haspopup="menu"]');
  assert.ok(trigger);
  await act(async () => trigger.click());
  await waitForReactCondition(() => document.querySelector('[role="menuitem"]'), "header menu opens");
  return [...document.querySelectorAll('[role="menuitem"]')];
}
await test("spreadsheet sorting applies both directions and clears through the same menu contract", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  window.localStorage.clear();
  const changes = [];
  let closes = 0;
  function Header() {
    const [displayFilters, update] = React.useState({ order_by: "-created_at" });
    return React.createElement(HeaderColumn, {
      property: "priority",
      displayFilters,
      onClose: () => closes++,
      handleDisplayFilterUpdate: (data) => {
        changes.push(data.order_by);
        update(data);
      },
    });
  }
  await env.render(React.createElement(Header));
  let items = await openMenu(env);
  assert.equal(items.length, 2);
  await act(async () => items[0].click());
  assert.equal(changes.at(-1), SPREADSHEET_PROPERTY_DETAILS.priority.ascendingOrderKey);
  items = await openMenu(env);
  await act(async () => items[1].click());
  assert.equal(changes.at(-1), SPREADSHEET_PROPERTY_DETAILS.priority.descendingOrderKey);
  items = await openMenu(env);
  const clear = items.find((item) => item.textContent.includes("clear_sorting"));
  assert.ok(clear);
  await act(async () => clear.click());
  assert.equal(changes.at(-1), "-created_at");
  assert.ok(closes >= 3);
});
await test("Kanban headers preserve grouped creation defaults, disablement and existing-item selection", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const fixture = globalThis.layoutHeaderFixture;
  fixture.route = { workspaceSlug: "workspace", projectId: "project" };
  const props = {
    column_id: "open",
    title: "Open",
    count: 2,
    sub_group_by: null,
    collapsedGroups: { group_by: [], sub_group_by: [] },
    handleCollapsedGroups: () => {},
    issuePayload: { state_id: "open" },
  };
  await env.render(React.createElement(HeaderGroupByCard, props));
  const create = env.container.querySelector('button[aria-label="Create work item"]');
  assert.equal(create.type, "button");
  await act(async () => create.click());
  assert.equal(fixture.modals.issue.isOpen, true);
  assert.deepEqual(fixture.modals.issue.data, { state_id: "open" });
  await act(async () => fixture.modals.issue.onClose());
  assert.equal(fixture.modals.issue.isOpen, false);
  await env.render(React.createElement(HeaderGroupByCard, { ...props, disableIssueCreation: true }));
  assert.equal(env.container.querySelector('button[aria-label="Create work item"]'), null);
  fixture.route = { ...fixture.route, moduleId: "module" };
  await env.render(React.createElement(HeaderGroupByCard, props));
  let items = await openMenu(env);
  await act(async () => items[1].click());
  assert.equal(fixture.modals.existing.isOpen, true);
  assert.deepEqual(fixture.modals.existing.searchParams, { module: "module" });
  await act(async () => fixture.modals.existing.handleClose());
  assert.equal(fixture.modals.existing.isOpen, false);
  items = await openMenu(env);
  await act(async () => items[0].click());
  assert.equal(fixture.modals.issue.isOpen, true);
});

async function registerGroupDialogContract(name, Component, route, searchParams) {
  await test(`${name} group dialogs retain epic defaults and add existing items only in their current scope`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const fixture = globalThis.layoutHeaderFixture;
    fixture.route = route;
    fixture.modals = {};
    const added = [];
    const props = {
      column_id: "open",
      groupID: "open",
      groupBy: "state",
      title: "Open",
      count: 2,
      sub_group_by: null,
      collapsedGroups: { group_by: [], sub_group_by: [] },
      handleCollapsedGroups: () => {},
      issuePayload: { state_id: "open", project_id: "project" },
      isEpic: true,
      canEditProperties: () => true,
      selectionHelpers: { isGroupSelected: () => "empty", isSelectionDisabled: false },
      addIssuesToView: async (ids) => {
        added.push(ids);
      },
    };
    await env.render(React.createElement(Component, props));
    let items = await openMenu(env);
    await act(async () => items.find((item) => item.textContent.includes("Create work item")).click());
    assert.equal(fixture.modals.epic.isOpen, true);
    assert.deepEqual(fixture.modals.epic.data, props.issuePayload);
    assert.equal(fixture.modals.issue, undefined, "an epic group uses the epic editor");
    await act(async () => fixture.modals.epic.onClose());
    assert.equal(fixture.modals.epic.isOpen, false);
    items = await openMenu(env);
    await act(async () => items.find((item) => item.textContent.includes("Add an existing work item")).click());
    assert.equal(fixture.modals.existing.isOpen, true);
    assert.deepEqual(fixture.modals.existing.searchParams, searchParams);
    await act(async () => fixture.modals.existing.handleOnSubmit([{ id: "first" }, { id: "second" }]));
    assert.deepEqual(added, [["first", "second"]]);
    await act(async () => fixture.modals.existing.handleClose());
    assert.equal(fixture.modals.existing.isOpen, false);
    fixture.route = { ...route, workspaceSlug: undefined };
    await env.render(React.createElement(Component, props));
    await act(async () => fixture.modals.existing.handleOnSubmit([{ id: "third" }]));
    assert.deepEqual(added, [["first", "second"]], "an incomplete route cannot dispatch an addition");
  });
}

await registerGroupDialogContract(
  "Kanban",
  HeaderGroupByCard,
  { workspaceSlug: "workspace", projectId: "project", cycleId: "cycle" },
  { cycle: true }
);
await registerGroupDialogContract(
  "List",
  ListGroupByCard,
  { workspaceSlug: "workspace", projectId: "project", moduleId: "module" },
  { module: "module" }
);

await test("the List header creation control is named, focusable and opens the issue editor", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const fixture = globalThis.layoutHeaderFixture;
  fixture.route = { workspaceSlug: "workspace", projectId: "project" };
  fixture.modals = {};
  await env.render(
    React.createElement(ListGroupByCard, {
      groupID: "open",
      groupBy: "state",
      title: "Open",
      count: 1,
      issuePayload: { state_id: "open" },
      canEditProperties: () => false,
      selectionHelpers: { isGroupSelected: () => "empty", isSelectionDisabled: false },
      handleCollapsedGroups: () => {},
    })
  );
  const create = env.container.querySelector('button[aria-label="Create work item"]');
  assert.ok(create, "creation must be available to keyboard users");
  assert.equal(create.type, "button");
  create.focus();
  assert.equal(document.activeElement, create);
  await act(async () => create.click());
  assert.equal(fixture.modals.issue.isOpen, true);
  assert.deepEqual(fixture.modals.issue.data, { state_id: "open" });
});

await test("the List group heading is a focusable control that toggles its own group", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  globalThis.layoutHeaderFixture.route = { workspaceSlug: "workspace", projectId: "project" };
  const toggled = [];
  await env.render(
    React.createElement(ListGroupByCard, {
      groupID: "open",
      groupBy: "state",
      title: "Open",
      count: 1,
      issuePayload: {},
      disableIssueCreation: true,
      canEditProperties: () => false,
      selectionHelpers: { isGroupSelected: () => "empty", isSelectionDisabled: false },
      handleCollapsedGroups: (id) => toggled.push(id),
    })
  );
  const heading = [...env.container.querySelectorAll("button")].find((node) => node.textContent.includes("Open"));
  assert.ok(heading, "the group heading must be available to keyboard users");
  heading.focus();
  assert.equal(document.activeElement, heading);
  await act(async () => heading.click());
  assert.deepEqual(toggled, ["open"]);
});
