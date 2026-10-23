import assert from "node:assert/strict";
import test from "node:test";
import { webRequire } from "./helpers/react-dom.mjs";
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
