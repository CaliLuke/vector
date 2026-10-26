import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createPermissionQueryClient } from "./helpers/permission-query-fixture.mjs";

registerWebModuleHooks();
const { webRequire } = await import("./helpers/react.mjs");
const { MutationObserver, QueryObserver } = webRequire("@tanstack/react-query");
const { intakeKeys, intakeListOptions, intakeDetailOptions, intakeMutationOptions } =
  await import("../../apps/web/core/queries/intake.ts");
const { defaultIntakePreferences, intakeQueryParams, selectIntakeRows } =
  await import("../../apps/web/helpers/intake-preferences.ts");
const { InboxIssueService } = await import("../../apps/web/core/services/inbox/inbox-issue.service.ts");
const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");

const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "workspace",
  membershipRole: 15,
  membershipRevision: "v1",
  projectId: "project",
  projectMembershipRole: 20,
  projectRole: 20,
};
const row = {
  id: "inbox",
  issue: { id: "item", project_id: "project", name: "Original" },
  status: -2,
  snoozed_till: null,
};
const page = { results: [row], total_results: 1, next_page_results: false, next_cursor: "" };
const preferences = defaultIntakePreferences();
const fixture = (t) => createPermissionQueryClient(t, { ...scope, workspaceRole: scope.membershipRole });

await test("identical native intake list reads deduplicate and propagate cancellation", async (t) => {
  const { client } = await fixture(t);
  const pending = Promise.withResolvers();
  let requests = 0;
  let signal;
  t.mock.method(InboxIssueService.prototype, "list", (_workspace, _project, params, options) => {
    requests++;
    signal = options.signal;
    assert.equal(params.cursor, "10:0:0");
    assert.equal(params.status, "-2");
    return pending.promise;
  });
  const options = intakeListOptions(client, scope, preferences);
  const first = client.fetchInfiniteQuery(options);
  const second = client.fetchInfiniteQuery(options);
  assert.equal(requests, 1);
  const rejected = Promise.all([assert.rejects(first), assert.rejects(second)]);
  await client.cancelQueries({ queryKey: options.queryKey, exact: true });
  assert.equal(signal.aborted, true);
  pending.resolve(page);
  await rejected;
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("a replaced permission query prevents a late intake detail response entering the cache", async (t) => {
  const { client, rolesKey } = await fixture(t);
  const pending = Promise.withResolvers();
  t.mock.method(InboxIssueService.prototype, "retrieve", () => pending.promise);
  const options = intakeDetailOptions(client, scope, "item");
  const rejected = assert.rejects(client.fetchQuery(options), /access changed/);
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { project: 20 });
  pending.resolve(row);
  await rejected;
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("intake reads reject records belonging to another project", async (t) => {
  const { client } = await fixture(t);
  t.mock.method(InboxIssueService.prototype, "retrieve", async () => ({
    ...row,
    issue: { ...row.issue, project_id: "other" },
  }));
  await assert.rejects(client.fetchQuery(intakeDetailOptions(client, scope, "item")), /outside the requested project/);
  assert.equal(client.getQueryData(intakeKeys.detail(scope, "item")), undefined);
});

await test("accepted intake mutations cancel earlier detail reads and preserve the canonical receipt", async (t) => {
  const { client } = await fixture(t);
  const key = intakeKeys.detail(scope, "item");
  client.setQueryData(key, row);
  const pending = Promise.withResolvers();
  let signal;
  t.mock.method(InboxIssueService.prototype, "retrieve", (_workspace, _project, _issue, options) => {
    signal = options.signal;
    return pending.promise;
  });
  const read = client
    .fetchQuery({ ...intakeDetailOptions(client, scope, "item"), staleTime: 0 })
    .catch((error) => error);
  const saved = { ...row, status: 1 };
  t.mock.method(InboxIssueService.prototype, "update", async () => saved);
  const effects = [];
  const mutation = new MutationObserver(
    client,
    intakeMutationOptions(client, scope, () => ({
      beforeWrite: async () => {},
      accepted: async (receipt) => {
        effects.push(receipt);
      },
    }))
  );
  const receipt = await mutation.mutate({ kind: "status", issueId: "item", data: { status: 1 } });
  pending.resolve(row);
  await read;
  assert.equal(signal.aborted, true);
  assert.deepEqual(client.getQueryData(key), saved);
  assert.equal(receipt.saved.status, 1);
  assert.equal(effects.length, 1);
});

await test("an accepted response from an earlier session cannot update new intake state or project counts", async (t) => {
  const { client } = await fixture(t);
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  let effects = 0;
  t.mock.method(InboxIssueService.prototype, "update", () => {
    started.resolve();
    return pending.promise;
  });
  const mutation = new MutationObserver(
    client,
    intakeMutationOptions(client, scope, () => ({
      beforeWrite: async () => {},
      accepted: async () => {
        effects++;
      },
    }))
  );
  const write = mutation.mutate({ kind: "status", issueId: "item", data: { status: 1 } });
  await started.promise;
  advanceIdentitySession(client);
  const newKey = intakeKeys.detail({ ...scope, sessionGeneration: 1 }, "item");
  client.setQueryData(newKey, { ...row, issue: { ...row.issue, name: "Current session" } });
  pending.resolve({ ...row, status: 1 });
  await write;
  assert.equal(effects, 0);
  assert.equal(client.getQueryData(newKey).issue.name, "Current session");
  assert.equal(client.getQueryData(newKey).status, -2);
});

await test("intake filter projections preserve explicit dates, closed tabs, and expired snoozes", () => {
  const dated = { ...preferences, filters: { status: [-2], created_at: ["2026-10-01;after"] } };
  assert.equal(intakeQueryParams(dated).created_at, "2026-10-01;after");
  const snoozed = { ...row, status: 0, snoozed_till: "2026-10-01T00:00:00Z" };
  const future = { ...snoozed, issue: { ...row.issue, id: "future" }, snoozed_till: "2030-01-01T00:00:00Z" };
  assert.deepEqual(selectIntakeRows([snoozed, future], preferences, Date.parse("2026-10-25")), [snoozed]);
  assert.deepEqual(selectIntakeRows([row], { ...preferences, filters: { status: [] } }, Date.now()), []);
  assert.deepEqual(defaultIntakePreferences("closed").filters.status, [1, -1, 2]);
});

await test("accepted intake status keeps an active detail canonical without a follow-up read", async (t) => {
  const { client } = await fixture(t);
  const key = intakeKeys.detail(scope, "item");
  client.setQueryData(key, row);
  let requests = 0;
  t.mock.method(InboxIssueService.prototype, "retrieve", async () => {
    requests++;
    throw new Error("Follow-up unavailable");
  });
  const observer = new QueryObserver(client, { ...intakeDetailOptions(client, scope, "item"), staleTime: Infinity });
  const unsubscribe = observer.subscribe(() => {});
  t.after(unsubscribe);
  const saved = { ...row, status: 1 };
  t.mock.method(InboxIssueService.prototype, "update", async () => saved);
  const mutation = new MutationObserver(
    client,
    intakeMutationOptions(client, scope, () => ({
      beforeWrite: async () => {},
      accepted: async () => {},
    }))
  );
  await mutation.mutate({ kind: "status", issueId: "item", data: { status: 1 } });
  assert.equal(requests, 0);
  assert.equal(observer.getCurrentResult().status, "success");
  assert.deepEqual(observer.getCurrentResult().data, saved);
});
