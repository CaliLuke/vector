import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
registerWebTestModules({ "@plane/i18n": "export const useTranslation=()=>({t:key=>key});" });
const { GroupPagination } = await import("../../apps/web/core/components/issues/issue-layouts/group-pagination.tsx");
const { groupDropPolicy } = await import("../../apps/web/core/components/issues/issue-layouts/group-drag-overlay.tsx");

for (const layout of ["list", "kanban"]) {
  void test(`${layout} manual pagination follows server exhaustion and disables repeat requests while pending`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    let requests = 0;
    let submissions = 0;
    const props = {
      layout,
      manual: true,
      loadedCount: 2,
      totalCount: 3,
      isPaginating: false,
      loadMore: () => {
        requests++;
      },
      loader: React.createElement("span", { "data-loader": true }),
      sentinel: null,
    };
    const render = (changes = {}) =>
      environment.render(
        React.createElement(
          "form",
          {
            onSubmit: (e) => {
              e.preventDefault();
              submissions++;
            },
          },
          React.createElement(GroupPagination, { ...props, ...changes })
        )
      );
    await render();
    const button = environment.container.querySelector("button");
    assert.equal(button?.type, "button");
    await act(async () => button.click());
    assert.equal(requests, 1);
    assert.equal(submissions, 0);
    await render({ isPaginating: true });
    assert.equal(environment.container.querySelector("button"), null);
    assert.equal(environment.container.querySelectorAll("[data-loader]").length, 1);
    await render({ nextPageResults: false });
    assert.equal(environment.container.querySelector("form").textContent, "");
    assert.equal(environment.container.querySelector("[data-loader]"), null);
    await render({ loadedCount: 3 });
    assert.equal(environment.container.querySelector("button"), null);
    await render({ loadedCount: 3, nextPageResults: true });
    assert.ok(environment.container.querySelector("button"), "server continuation overrides the current count");
  });
  void test(`${layout} automatic pagination retains a live observer sentinel and removes it when exhausted`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const sentinel = React.createRef();
    const props = {
      layout,
      manual: false,
      loadedCount: 0,
      totalCount: 2,
      isPaginating: false,
      loadMore: () => assert.fail("automatic loading belongs to the observer"),
      loader: React.createElement("span", { "data-loader": true }),
      sentinel: React.createElement("span", { ref: sentinel, "data-sentinel": true }),
    };
    await environment.render(React.createElement(GroupPagination, props));
    assert.equal(environment.container.querySelectorAll("[data-loader]").length, 2);
    assert.ok(sentinel.current?.isConnected);
    assert.equal(environment.container.querySelector("button"), null);
    await environment.render(React.createElement(GroupPagination, { ...props, nextPageResults: false }));
    assert.equal(sentinel.current, null);
    assert.equal(environment.container.textContent, "");
  });
}

await test("group drop policy distinguishes ordering overlays from forbidden moves", () => {
  assert.deepEqual(groupDropPolicy("sort_order", false, undefined), {
    isDropDisabled: false,
    canOverlayBeVisible: false,
  });
  assert.deepEqual(groupDropPolicy("name", false, false), { isDropDisabled: false, canOverlayBeVisible: true });
  assert.deepEqual(groupDropPolicy(undefined, false, false), { isDropDisabled: false, canOverlayBeVisible: true });
  for (const [workflow, group] of [
    [true, false],
    [false, true],
    [true, true],
  ]) {
    assert.deepEqual(groupDropPolicy("sort_order", workflow, group), {
      isDropDisabled: true,
      canOverlayBeVisible: true,
    });
  }
});

const { canDragDropForGrouping } = await import("../../apps/web/core/providers/issue-layout-ui-state.tsx");
await test("group dragging permits ungrouped lists and requires movable axes for Kanban", () => {
  assert.equal(canDragDropForGrouping(null, undefined, { allowUngrouped: true }), true);
  assert.equal(canDragDropForGrouping(undefined, undefined), false);
  assert.equal(canDragDropForGrouping("state", undefined), true);
  assert.equal(canDragDropForGrouping("state", "priority"), true);
  assert.equal(canDragDropForGrouping("created_by", undefined), false);
  assert.equal(canDragDropForGrouping("state", "created_by"), false);
});

const { issueLayoutCapabilities, isIssuePropertyEditable } =
  await import("../../apps/web/core/components/issues/issue-layouts/utils.tsx");
const { EIssuesStoreType } = await import("../../packages/types/dist/index.mjs");
await test("layout capabilities preserve archived-list and subscribed-profile restrictions", () => {
  assert.deepEqual(issueLayoutCapabilities("list", EIssuesStoreType.ARCHIVED, {}), {
    enableQuickAdd: false,
    enableIssueCreation: false,
    enableInlineEditing: false,
  });
  for (const layout of ["list", "kanban"]) {
    assert.deepEqual(issueLayoutCapabilities(layout, EIssuesStoreType.PROJECT, {}), {
      enableQuickAdd: true,
      enableIssueCreation: true,
      enableInlineEditing: true,
    });
    assert.deepEqual(
      issueLayoutCapabilities(layout, EIssuesStoreType.PROFILE, { kind: "profile", view: "subscribed" }),
      { enableQuickAdd: false, enableIssueCreation: false, enableInlineEditing: true }
    );
    assert.deepEqual(issueLayoutCapabilities(layout, EIssuesStoreType.PROFILE, { kind: "profile", view: "assigned" }), {
      enableQuickAdd: false,
      enableIssueCreation: true,
      enableInlineEditing: true,
    });
  }
});
await test("per-project editing decisions override route permissions while read-only views stay read-only", () => {
  assert.equal(
    isIssuePropertyEditable("allowed", (id) => id === "allowed", true, false),
    true
  );
  assert.equal(
    isIssuePropertyEditable("denied", () => false, true, true),
    false
  );
  assert.equal(
    isIssuePropertyEditable("allowed", () => true, false, true),
    false
  );
  assert.equal(
    isIssuePropertyEditable(undefined, () => assert.fail("no project to authorize"), true, true),
    true
  );
  assert.equal(isIssuePropertyEditable("project", undefined, true, false), false);
  assert.equal(isIssuePropertyEditable("project", undefined, undefined, true), false);
});

const { handleGroupDragDrop, getBlockViewDetails } =
  await import("../../apps/web/core/components/issues/issue-layouts/utils.tsx");
await test("group moves preserve unrelated labels, reconcile both axes and propagate failed writes", async () => {
  const issue = {
    id: "item",
    project_id: "project",
    label_ids: ["old", "kept", "new"],
    state_id: "open",
    sort_order: 10,
  };
  const writes = [];
  const move = (
    source,
    destination,
    group = "labels",
    subgroup = "state",
    update = async (...args) => writes.push(args)
  ) =>
    handleGroupDragDrop(
      source,
      destination,
      (id) => (id === "item" ? issue : undefined),
      () => [],
      update,
      group,
      subgroup
    );
  const source = { id: "item", groupId: "old", subGroupId: "open" };
  const destination = { groupId: "new", subGroupId: "closed" };
  await move(source, destination);
  assert.deepEqual(writes, [
    [
      "project",
      "item",
      {
        id: "item",
        project_id: "project",
        sort_order: 65535,
        label_ids: ["kept", "new"],
        state_id: "closed",
      },
      {
        label_ids: { ADD: ["new"], REMOVE: ["old"] },
        state_id: { ADD: ["closed"], REMOVE: ["open"] },
      },
    ],
  ]);
  assert.deepEqual(issue.label_ids, ["old", "kept", "new"], "moving a card never mutates cached source data");
  await move(source, { groupId: "None", subGroupId: "None" });
  assert.deepEqual(writes[1][2].label_ids, ["kept", "new"]);
  assert.equal(writes[1][2].state_id, null);
  assert.deepEqual(writes[1][3], { label_ids: { ADD: [], REMOVE: ["old"] }, state_id: { ADD: [], REMOVE: ["open"] } });
  await move(source, { groupId: "old", subGroupId: "open" });
  assert.deepEqual(writes[2][3], {}, "reordering inside the same groups has no membership effects");
  await move({ id: "missing", groupId: "old", subGroupId: "open" }, destination);
  await move({ id: "item", groupId: "old" }, destination);
  assert.equal(writes.length, 3, "incomplete or missing sources must not write");
  await assert.rejects(
    move(source, destination, "labels", "state", async () => {
      throw new Error("offline");
    }),
    /offline/
  );
});

await test("timeline blocks distinguish missing dates, one-ended gradients and complete date ranges", () => {
  for (const block of [undefined, null, {}, { start_date: null, target_date: null }]) {
    assert.deepEqual(getBlockViewDetails(block, "red"), { message: undefined, blockStyle: { backgroundColor: "red" } });
  }
  const start = getBlockViewDetails({ start_date: "2026-10-01" }, "red");
  const end = getBlockViewDetails({ target_date: "2026-10-02" }, "red");
  const complete = getBlockViewDetails({ start_date: "2026-10-01", target_date: "2026-10-02" }, "red");
  assert.match(start.message, /^From .+/);
  assert.match(start.blockStyle.maskImage, /to right/);
  assert.match(end.message, /^Till .+/);
  assert.match(end.blockStyle.maskImage, /to left/);
  assert.equal(complete.message, `${start.message.slice(5)} to ${end.message.slice(5)}`);
  assert.deepEqual(complete.blockStyle, { backgroundColor: "red" });
});
