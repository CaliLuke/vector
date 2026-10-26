import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { getModulePeekHref } = await import("../../apps/web/core/components/modules/module-peek-navigation.ts");

await test("module peek navigation preserves unrelated query parameters while toggling the selected module", () => {
  assert.equal(
    getModulePeekHref("/workspace/projects/project/modules", "filter=active&peekModule=module-a", "module-a"),
    "/workspace/projects/project/modules?filter=active"
  );
  assert.equal(
    getModulePeekHref("/workspace/projects/project/modules", "filter=active&peekModule=module-a", "module-b"),
    "/workspace/projects/project/modules?filter=active&peekModule=module-b"
  );
});
