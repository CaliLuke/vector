import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";

const fixture = { calls: [], saved: [] };
globalThis.propertyRestoreFixture = fixture;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL?.endsWith("/use-property-restore.ts") && specifier === "./use-published-item-edit") {
      const source = `export const usePublishedItemEdit=(scope,onSaved)=>({source:{data:globalThis.propertyRestoreFixture.source},conflict:globalThis.propertyRestoreFixture.conflict??false,save:async(command)=>{globalThis.propertyRestoreFixture.calls.push(command);onSaved(command,false)}});`;
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    return nextResolve(specifier, context);
  },
});
const { usePropertyRestore } = await import("../../apps/web/core/components/work-item-types/use-property-restore.ts");
const scope = { workspaceSlug: "acme", projectId: "p1", issueId: "item-1" };
const version = { id: "revision-1", property_id: "field-1", value: "Historical value" };

await test("property restore submits its immutable revision through the published item save contract", async () => {
  fixture.calls = [];
  fixture.saved = [];
  fixture.source = { schema: { revision: 7 }, source: { source_version: "fingerprint-1" } };
  fixture.conflict = false;
  const hook = usePropertyRestore(
    scope,
    version,
    (...args) => fixture.saved.push(args),
    () => assert.fail("access lost")
  );

  await hook.restore();

  assert.deepEqual(fixture.calls, [
    {
      mode: "edit",
      item_id: "item-1",
      project_id: "p1",
      revision: 7,
      expected_source_version: "fingerprint-1",
      changes: {},
      restore: { property_id: "field-1", version_id: "revision-1" },
    },
  ]);
  assert.deepEqual(fixture.saved, [["Revision restored.", false]]);
});

await test("property restore does not submit without a current source or while conflicted", async () => {
  fixture.calls = [];
  fixture.source = undefined;
  fixture.conflict = false;
  const unavailable = usePropertyRestore(
    scope,
    version,
    () => {},
    () => {}
  );
  await unavailable.restore();
  assert.equal(fixture.calls.length, 0);

  fixture.source = { schema: { revision: 7 }, source: { source_version: "fingerprint-1" } };
  fixture.conflict = true;
  const conflicted = usePropertyRestore(
    scope,
    version,
    () => {},
    () => {}
  );
  await conflicted.restore();
  assert.equal(fixture.calls.length, 0);
});

delete globalThis.propertyRestoreFixture;
