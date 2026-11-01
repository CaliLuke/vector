import assert from "node:assert/strict";
import test from "node:test";
import { webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const { attachInstruction } = webRequire("@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item");
registerWebModuleHooks();
const { getSourceFromDropPayload, getDestinationFromDropPayload } =
  await import("../../apps/web/helpers/issue-list/drop-payload.ts");
const payload = (initial, current = [], id = "source") => ({
  source: { data: { id } },
  location: {
    initial: { dropTargets: initial.map((data) => ({ data })) },
    current: { dropTargets: current.map((data) => ({ data })) },
  },
});

void test("list and Kanban drops preserve their supported group identities", () => {
  assert.deepEqual(getSourceFromDropPayload(payload([{ type: "COLUMN", groupId: "list" }])), {
    groupId: "list",
    columnId: undefined,
    subGroupId: undefined,
    id: "source",
  });
  const column = { type: "COLUMN", groupId: "group", subGroupId: "subgroup", columnId: "group__subgroup" };
  assert.deepEqual(
    getSourceFromDropPayload(payload([{ type: "IGNORE" }, { type: "COLUMN", groupId: "inner" }, column])),
    {
      groupId: "group",
      subGroupId: "subgroup",
      columnId: "group__subgroup",
      id: "source",
    }
  );
  assert.deepEqual(
    getDestinationFromDropPayload(payload([], [{ type: "DELETE", groupId: "trash", columnId: "trash" }])),
    {
      groupId: "trash",
      columnId: "trash",
      subGroupId: undefined,
      id: undefined,
      canAddIssueBelow: false,
    }
  );
});

void test("malformed drop identities are rejected before they can reach work-item mutations", () => {
  for (const column of [
    undefined,
    null,
    { type: "COLUMN" },
    { type: "COLUMN", groupId: 42 },
    { type: "COLUMN", groupId: "" },
    { type: "COLUMN", groupId: "group", subGroupId: [] },
    { type: "COLUMN", groupId: "group", columnId: 42 },
  ]) {
    assert.equal(getSourceFromDropPayload(payload([column])), undefined);
    assert.equal(getDestinationFromDropPayload(payload([], [column])), undefined);
  }
  const column = { type: "COLUMN", groupId: "group" };
  for (const id of [null, "", 42, {}]) {
    assert.equal(getSourceFromDropPayload(payload([column], [], id)), undefined);
    assert.equal(getDestinationFromDropPayload(payload([], [column, { type: "ISSUE", id }])), undefined);
  }
  assert.equal(getSourceFromDropPayload({ source: {}, location: {} }), undefined);
  assert.equal(getDestinationFromDropPayload({ location: {} }), undefined);
});

void test("destination item placement retains the hitbox instruction and optional subgroup", () => {
  const column = { type: "COLUMN", groupId: "group", subGroupId: null };
  const issue = attachInstruction(
    { type: "ISSUE", id: "target" },
    {
      element: { getBoundingClientRect: () => ({ top: 0, bottom: 100, height: 100, left: 0, right: 100 }) },
      input: { clientX: 50, clientY: 99 },
      currentLevel: 0,
      indentPerLevel: 20,
      mode: "standard",
    }
  );
  assert.deepEqual(getDestinationFromDropPayload(payload([], [issue, column])), {
    groupId: "group",
    columnId: undefined,
    subGroupId: undefined,
    id: "target",
    canAddIssueBelow: true,
  });
  assert.equal(
    getDestinationFromDropPayload(payload([], [{ type: "ISSUE", id: "target" }, column])).canAddIssueBelow,
    false
  );
});
