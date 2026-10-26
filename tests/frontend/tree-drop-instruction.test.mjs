import assert from "node:assert/strict";
import test from "node:test";
import { webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const { attachInstruction } = webRequire("@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item");
registerWebModuleHooks();
const { getTreeDropInstruction } = await import("../../apps/web/core/lib/tree-drop-instruction.ts");

function target({ y = 50, isChild = false, isGroup = false, block = [] } = {}) {
  return {
    data: attachInstruction(
      { id: "target", parentId: null, isChild, isGroup },
      {
        element: { getBoundingClientRect: () => ({ top: 0, bottom: 100, height: 100, left: 0, right: 100 }) },
        input: { clientX: 50, clientY: y },
        currentLevel: 0,
        indentPerLevel: 20,
        mode: "standard",
        block,
      }
    ),
  };
}

const item = { data: { id: "item", parentId: null, isGroup: false, isChild: false } };
const group = { data: { id: "group", parentId: null, isGroup: true, isChild: false } };
const location = { current: { dropTargets: [] } };

await test("tree drops retain above, below, and child placement from the hitbox", () => {
  assert.equal(getTreeDropInstruction(target({ y: 10 }), item, location), "reorder-above");
  assert.equal(getTreeDropInstruction(target({ y: 90 }), item, location), "reorder-below");
  assert.equal(getTreeDropInstruction(target(), item, location), "make-child");
});

await test("groups cannot be nested by a single-target drop", () => {
  assert.equal(getTreeDropInstruction(target(), group, location), "reorder-above");
});

await test("blocked child placement respects the target hierarchy", () => {
  assert.equal(
    getTreeDropInstruction(target({ isChild: true, block: ["make-child"] }), item, location),
    "reorder-above"
  );
  assert.equal(getTreeDropInstruction(target({ block: ["make-child"] }), item, location), "make-child");
});

await test("overlapping parent and child targets retain the parent grouping instruction", () => {
  const parent = target({ isGroup: true });
  const nestedLocation = { current: { dropTargets: [target({ isChild: true }), parent] } };
  assert.equal(getTreeDropInstruction(parent, item, nestedLocation), "make-child");
});

await test("a missing drag payload produces no instruction", () => {
  assert.equal(getTreeDropInstruction(target(), {}, location), undefined);
  assert.equal(getTreeDropInstruction({}, item, location), undefined);
});

await test("foreign drag payloads cannot produce a tree placement", () => {
  assert.equal(getTreeDropInstruction(target(), { data: { id: 123, isGroup: false } }, location), undefined);
  assert.equal(
    getTreeDropInstruction({ data: { id: "target", parentId: [], isGroup: true, isChild: false } }, item, location),
    undefined
  );
});

const { treeDropPositionTarget } = await import("../../apps/web/helpers/tree-drop-position.ts");
const rowElement = { getBoundingClientRect: () => ({ top: 0, bottom: 100, height: 100, left: 0, right: 100 }) };

await test("flat row hitboxes keep the final lower edge and regular row placement", () => {
  const positions = [];
  const last = treeDropPositionTarget("last", true, (position) => positions.push(position));
  const row = treeDropPositionTarget("row", false, (position) => positions.push(position));
  const below = last.getData({ element: rowElement, input: { clientX: 50, clientY: 99 } });
  const above = row.getData({ element: rowElement, input: { clientX: 50, clientY: 1 } });
  last.onDrag({ self: { data: below } });
  row.onDrag({ self: { data: above } });
  row.onDragLeave();
  assert.deepEqual(positions, ["DRAG_BELOW", "DRAG_OVER", undefined]);
  assert.equal(below.id, "last");
  assert.equal(above.id, "row");
});

await test("flat row drops clear feedback and retain source, destination, and placement", () => {
  const positions = [];
  const drops = [];
  const source = { data: { id: "source" }, element: { id: "dragged-row" } };
  const rowTarget = treeDropPositionTarget(
    "last",
    true,
    (position) => positions.push(position),
    (...args) => drops.push(args)
  );
  const data = rowTarget.getData({ element: rowElement, input: { clientX: 50, clientY: 99 } });
  rowTarget.onDrop({ self: { data }, source });
  rowTarget.onDrop({ self: { data: { id: "unrecognized" } }, source });
  assert.deepEqual(drops, [["source", "last", true, source.element]]);
  assert.deepEqual(positions, [undefined, undefined]);
});
