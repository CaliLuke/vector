import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { parentCatalogueFixture, parentScope, seedParentListPreferences } from "./helpers/parent-catalogue-fixture.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";

registerWebModuleHooks();
const nativeForms = observeNativeForms(["apps/web/core/components/cycles/transfer-issues-modal.tsx"]);
const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");

async function setup(t, transferRequest, refreshIssues) {
  const fixture = await createQueryRouterFixture(t, "/acme/projects/p1/cycles/source/");
  const { TransferIssuesModal } = await import("../../apps/web/core/components/cycles/transfer-issues-modal.tsx");
  let closed = 0;
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(fixture.client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole: 15,
    projectRoles: { p1: 20 },
  });
  const source = { id: "source", project_id: "p1", name: "Source cycle", status: "CURRENT" };
  const target = { id: "target", project_id: "p1", name: "Target cycle", status: "UPCOMING" };
  let initialRead = true;
  t.mock.method(CycleService.prototype, "getWithParams", async () => {
    if (initialRead) initialRead = false;
    else await refreshIssues();
    return [source, target];
  });
  await seedParentListPreferences(t, fixture.client, "cycle", source.id, {
    richFilters: {},
    displayFilters: { layout: "list" },
    displayProperties: {},
  });
  t.mock.method(CycleService.prototype, "transferIssues", transferRequest);
  const { ParentIssuePreferencesProvider } = await import("../../apps/web/core/providers/parent-issue-preferences.tsx");
  await fixture.render(
    React.createElement(
      ParentIssuePreferencesProvider,
      { domain: "cycle", workspaceSlug: "acme", projectId: "p1", parentId: source.id },
      React.createElement(TransferIssuesModal, {
        isOpen: true,
        cycleId: source.id,
        workspaceSlug: "acme",
        projectId: "p1",
        handleClose: () => closed++,
      })
    )
  );
  return { closed: () => closed };
}
await test("cycle refresh retries do not repeat a successful transfer", async (t) => {
  const pending = Promise.withResolvers();
  let transfers = 0,
    refreshes = 0;
  const fixture = await setup(
    t,
    async () => {
      transfers++;
      return pending.promise;
    },
    async () => {
      refreshes++;
      if (refreshes === 1) throw new Error("offline");
    }
  );
  const targets = document.querySelectorAll("button[aria-pressed]");
  assert.equal(targets.length, 1, "the source cycle cannot be selected");
  await act(async () => targets[0].click());
  assert.equal(transfers, 1, "the accepted transfer request should start");
  await settleQuery();
  assert.equal(document.querySelector("fieldset").disabled, true);
  assert.equal(document.querySelector('[aria-label="Close transfer"]').disabled, true);
  assert.equal(fixture.closed(), 0);
  await act(async () => pending.resolve({}));
  await settleQuery();
  assert.match(
    [...document.querySelectorAll('[role="alert"]')].map((element) => element.textContent).join(" "),
    /were transferred/
  );
  assert.equal(
    nativeForms.getForms("apps/web/core/components/cycles/transfer-issues-modal.tsx").at(-1).state.isSubmitSuccessful,
    true
  );
  assert.equal(document.querySelector("fieldset").disabled, true, "the completed destination cannot be changed");
  await act(async () => document.querySelector('button[type="submit"]').click());
  assert.equal(transfers, 1);
  assert.equal(fixture.closed(), 1);
});
await test("a rejected cycle transfer keeps the selected destination available for retry", async (t) => {
  let attempts = 0;
  const fixture = await setup(
    t,
    async () => {
      attempts++;
      if (attempts === 1) throw new Error("offline");
    },
    async () => {}
  );
  await act(async () => document.querySelector("button[aria-pressed]").click());
  assert.equal(attempts, 1, "the first transfer request should start");
  assert.equal(
    nativeForms.getForms("apps/web/core/components/cycles/transfer-issues-modal.tsx").at(-1).state.isSubmitSuccessful,
    false
  );
  assert.match(document.querySelector('[role="alert"]').textContent, /Unable to transfer/);
  assert.equal(document.querySelector("fieldset").disabled, false);
  assert.equal(document.querySelector("button[aria-pressed]").getAttribute("aria-pressed"), "true");
  assert.equal(fixture.closed(), 0);
  await act(async () => document.querySelector('button[type="submit"]').click());
  assert.equal(
    nativeForms.getForms("apps/web/core/components/cycles/transfer-issues-modal.tsx").at(-1).state.isSubmitSuccessful,
    true
  );
  assert.equal(attempts, 2);
  assert.equal(fixture.closed(), 1);
});
await test("failed cycle progress refresh releases its loading state", async (t) => {
  const { client } = await parentCatalogueFixture(t);
  const { cycleProgressOptions } = await import("../../apps/web/core/queries/cycle-modules.ts");
  t.mock.method(CycleService.prototype, "workspaceActiveCyclesProgress", async () => {
    throw new Error("offline");
  });
  const options = cycleProgressOptions(parentScope, "cycle");
  await assert.rejects(client.fetchQuery(options), /offline/);
  const result = client.getQueryState(options.queryKey);
  assert.equal(result.status, "error");
  assert.equal(result.fetchStatus, "idle");
});

nativeForms.restore();
