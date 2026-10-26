import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

registerWebTestModules({
  "@/components/stickies/provider": "export const useStickies = () => ({ stickies: new Map(), stickyIds: [] });",
  "@plane/propel/toast":
    "export const TOAST_TYPE = { SUCCESS: 'success', ERROR: 'error', WARNING: 'warning' }; export const setToast = value => globalThis.stickySessionToasts.push(value);",
});
const { useStickyOperations } = await import("../../apps/web/core/components/stickies/sticky/use-operations.tsx");
const { StickyService } = await import("../../apps/web/core/services/sticky.service.ts");
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");

async function operationFixture(t) {
  const fixture = await createQueryDomFixture(t);
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  globalThis.stickySessionToasts = [];
  let operations;
  function Harness() {
    const owner = useStickyOperations({ workspaceSlug: "acme", stickyId: "sticky-1" });
    React.useLayoutEffect(() => {
      operations = owner;
    }, [owner]);
    return null;
  }
  await fixture.render(React.createElement(Harness));
  return { ...fixture, operations };
}

await test("late sticky save failure cannot display feedback in a replacement session", async (t) => {
  const { client, operations } = await operationFixture(t);
  const started = Promise.withResolvers();
  const result = Promise.withResolvers();
  t.mock.method(StickyService.prototype, "updateSticky", () => {
    started.resolve();
    return result.promise;
  });
  let failure;
  await React.act(async () => {
    failure = assert.rejects(
      operations.stickyOperations.update("sticky-1", { description_html: "old draft" }),
      /offline/
    );
    await started.promise;
  });
  await React.act(async () => {
    advanceIdentitySession(client);
    client.setQueryData(identityKeys.currentUser(), { id: "bob" });
    result.reject(new Error("offline"));
    await failure;
  });
  assert.deepEqual(globalThis.stickySessionToasts, []);
  assert.equal(operations.isCurrent(), false);
});

await test("late sticky deletion success cannot display feedback in a replacement session", async (t) => {
  const { client, operations } = await operationFixture(t);
  const started = Promise.withResolvers();
  const result = Promise.withResolvers();
  t.mock.method(StickyService.prototype, "deleteSticky", () => {
    started.resolve();
    return result.promise;
  });
  let deletion;
  await React.act(async () => {
    deletion = operations.stickyOperations.remove("sticky-1");
    await started.promise;
  });
  await React.act(async () => {
    advanceIdentitySession(client);
    client.setQueryData(identityKeys.currentUser(), { id: "bob" });
    result.resolve();
    await deletion;
  });
  assert.deepEqual(globalThis.stickySessionToasts, []);
});
