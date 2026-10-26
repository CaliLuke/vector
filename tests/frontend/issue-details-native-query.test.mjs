import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { seedIssueQueryScope } from "./helpers/issue-query-fixtures.mjs";

const { QueryClient, QueryClientProvider, useQuery } = await import("@tanstack/react-query");
const { EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL?.endsWith("/core/hooks/use-work-item-detail.ts")) {
      if (specifier === "./use-current-user")
        return createModuleStub(
          "export const useCurrentUser=()=>({data:globalThis.detailTestHookScope.accountId?{id:globalThis.detailTestHookScope.accountId}:undefined,isSuccess:true,isError:false,refetch:async()=>{}});"
        );
      if (specifier === "./use-identity-generation")
        return createModuleStub(
          "export const useIdentityGeneration=()=>globalThis.detailTestHookScope.sessionGeneration;"
        );
      if (specifier === "./use-permissions")
        return createModuleStub(
          "export const useProjectPermissions=(workspaceSlug,projectId)=>{const scope=globalThis.detailTestHookScope;return {workspaceRole:scope.membershipRole,projectRole:scope.projectRole,membershipQuery:{data:{is_active:true,role:scope.membershipRole,updated_at:scope.membershipRevision},isSuccess:true,isPending:false,isError:false,refetch:async()=>{}},rolesQuery:{data:{[projectId]:scope.projectMembershipRole},isSuccess:true,isPending:false,isError:false,refetch:async()=>{}},isPending:false,isError:false,can:()=>true}};"
        );
    }
    if (specifier === "@/services/issue/issue.service")
      return createModuleStub(
        "export class IssueService { subIssues(...args) { return globalThis.detailTestFixture.subItems(...args); } retrieve(...args) { return globalThis.detailTestFixture.retrieve(...args); } retrieveWithIdentifier(...args) { return globalThis.detailTestFixture.retrieveByIdentifier(...args); } }"
      );
    return nextResolve(specifier, context);
  },
});

const { allSubItems, issueByIdentifierOptions, issueDetailKeys, issueDetailOptions, issueSubItemsOptions } =
  await import("../../apps/web/core/queries/issue-details.ts");
const { itemAccessOptions } = await import("../../apps/web/core/queries/work-item-access.ts");
const { captureIssueDetailReconciliation } = await import("../../apps/web/core/queries/issue-details.ts");
const { useWorkItemByIdentifier, useWorkItemSubItems } =
  await import("../../apps/web/core/hooks/use-work-item-detail.ts");
const { useQueryClient } = await import("@tanstack/react-query");

function setup(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  const scope = {
    accountId: "alice",
    sessionGeneration: 0,
    workspaceSlug: "acme",
    membershipRole: EUserWorkspaceRoles.ADMIN,
    membershipRevision: "v1",
    projectId: "project",
    projectMembershipRole: 5,
    projectRole: 20,
    serviceType: "issues",
  };
  seedIssueQueryScope(client, scope);
  return { client, scope };
}

function setItemAccessStatus(client, scope, issueId, denied = false) {
  const queryKey = itemAccessOptions({
    accountId: scope.accountId,
    generation: scope.sessionGeneration,
    workspaceSlug: scope.workspaceSlug,
    projectId: scope.projectId,
    issueId,
    serviceType: scope.serviceType,
  }).queryKey;
  client.setQueryData(queryKey, denied);
  return queryKey;
}

function installDetailHookScope(t, environment, scope) {
  globalThis.detailTestHookScope = scope;
  t.after(async () => {
    await environment.unmount();
    delete globalThis.detailTestHookScope;
  });
}

function deferred() {
  let resolve;
  const promise = new Promise((resolvePromise) => (resolve = resolvePromise));
  return { promise, resolve };
}

function IdentifierProbe() {
  const result = useWorkItemByIdentifier("acme", "PRJ", "7", "project");
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", { id: "identifier-result" }, result.issue?.id ?? "hidden"),
    React.createElement("output", { id: "identifier-denied" }, String(result.isDenied))
  );
}

function EditableIdentifierProbe({ onSave }) {
  const client = useQueryClient();
  const detail = useWorkItemByIdentifier("acme", "PRJ", "7", "project");
  const edit = async () => {
    const owner = captureIssueDetailReconciliation(client, detail.scope, ["issue"]);
    const acceptedReceipt = await Promise.resolve({ ...detail.issue, name: "Edited title" });
    return owner.reconcile(acceptedReceipt);
  };
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", { id: "edited-title" }, detail.issue?.name ?? "hidden"),
    React.createElement("button", { onClick: () => onSave(edit()) }, "Save edit")
  );
}

function InitialDetailProbe({ client, scope }) {
  const query = useQuery(issueDetailOptions(client, scope, "issue"));
  return React.createElement("output", { id: "initial-detail-result" }, query.data?.name ?? query.fetchStatus);
}

await test("a late detail response cannot publish into an exact replacement query", async (t) => {
  const { client, scope } = setup(t);
  const pending = deferred();
  globalThis.detailTestFixture = { retrieve: () => pending.promise };
  const queryKey = issueDetailKeys.detail(scope, "issue");
  const read = client.fetchQuery(issueDetailOptions(client, scope, "issue"));
  await new Promise((resolve) => setImmediate(resolve));
  client.removeQueries({ queryKey, exact: true });
  client.setQueryData(queryKey, { id: "replacement", project_id: scope.projectId });
  pending.resolve({ id: "issue", project_id: scope.projectId });
  await assert.rejects(read);
  assert.equal(client.getQueryData(queryKey).id, "replacement");
});

await test("sub-item reads reject responses after the exact query is replaced", async (t) => {
  const { client, scope } = setup(t);
  const pending = deferred();
  globalThis.detailTestFixture = { subItems: () => pending.promise };
  const queryKey = issueDetailKeys.subItems(scope, "parent");
  setItemAccessStatus(client, scope, "parent");
  const read = client.fetchQuery(issueSubItemsOptions(client, scope, "parent"));
  await new Promise((resolve) => setImmediate(resolve));
  client.removeQueries({ queryKey, exact: true });
  client.setQueryData(queryKey, { sub_issues: [{ id: "replacement", project_id: scope.projectId }] });
  pending.resolve({ sub_issues: [{ id: "child", project_id: scope.projectId, parent_id: "parent" }] });
  await assert.rejects(read);
  assert.equal(client.getQueryData(queryKey).sub_issues[0].id, "replacement");
});

await test("identifier reads reject a canonical record with a different sequence id", async (t) => {
  const { client, scope } = setup(t);
  globalThis.detailTestFixture = {
    retrieveByIdentifier: async () => ({
      id: "issue",
      project_id: scope.projectId,
      sequence_id: 8,
    }),
  };
  const options = issueByIdentifierOptions(client, scope, "PRJ", "7");
  await assert.rejects(client.fetchQuery(options), /different work item/);
});

await test("an item's denial marker hides cached by-identifier data without refetching", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { client, scope } = setup(t);
  installDetailHookScope(t, environment, scope);
  const record = {
    id: "issue",
    project_id: scope.projectId,
    sequence_id: 7,
  };
  const queryKey = issueDetailKeys.byIdentifier(scope, "PRJ", "7");
  client.setQueryData(queryKey, record);
  setItemAccessStatus(client, scope, record.id);
  await environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(IdentifierProbe)));
  assert.equal(environment.container.querySelector("#identifier-result").textContent, "issue");
  const key = setItemAccessStatus(client, scope, record.id);
  await React.act(async () => {
    client.setQueryData(key, true);
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  assert.equal(environment.container.querySelector("#identifier-denied").textContent, "true");
  assert.equal(environment.container.querySelector("#identifier-result").textContent, "hidden");
  assert.equal(client.getQueryState(queryKey)?.fetchStatus, "idle");
  assert.equal(client.getQueryState(queryKey)?.data.id, "issue");
});

await test("a successful UI edit updates the native by-identifier record from its accepted receipt", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { client, scope } = setup(t);
  installDetailHookScope(t, environment, scope);
  const record = {
    id: "issue",
    project_id: scope.projectId,
    sequence_id: 7,
    name: "Original title",
  };
  const queryKey = issueDetailKeys.byIdentifier(scope, "PRJ", "7");
  client.setQueryData(queryKey, record);
  setItemAccessStatus(client, scope, record.id);
  let editPromise;
  let editResult;
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(EditableIdentifierProbe, {
        onSave: (pending) => {
          editPromise = pending;
        },
      })
    )
  );
  assert.equal(environment.container.querySelector("#edited-title").textContent, "Original title");
  await React.act(async () => {
    environment.container.querySelector("button").click();
    editResult = await editPromise;
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  assert.equal(editResult, 1);
  assert.equal(environment.container.querySelector("#edited-title").textContent, "Edited title");
  assert.equal(client.getQueryData(queryKey).name, "Edited title");
});

await test("a reviewed edit receipt cannot replace an exact by-identifier query replaced during submit", async (t) => {
  const { client, scope } = setup(t);
  const queryKey = issueDetailKeys.byIdentifier(scope, "PRJ", "7");
  const original = {
    id: "issue",
    project_id: scope.projectId,
    sequence_id: 7,
    name: "Original title",
  };
  client.setQueryData(queryKey, original);
  setItemAccessStatus(client, scope, original.id);
  const owner = captureIssueDetailReconciliation(client, scope, [original.id]);
  const pending = deferred();
  const update = pending.promise.then((receipt) => owner.reconcile(receipt));
  client.removeQueries({ queryKey, exact: true });
  client.setQueryData(queryKey, { ...original, id: "replacement", name: "Replacement record" });
  pending.resolve({ ...original, name: "Saved title" });
  assert.equal(await update, 0);
  assert.equal(client.getQueryData(queryKey).id, "replacement");
  assert.equal(client.getQueryData(queryKey).name, "Replacement record");
});

await test("detail receipts leave caches from an earlier workspace membership untouched", async (t) => {
  const { client, scope } = setup(t);
  const issue = {
    id: "issue",
    project_id: scope.projectId,
    sequence_id: 7,
    name: "Old membership record",
  };
  const oldMembershipScope = {
    ...scope,
    membershipRole: EUserWorkspaceRoles.MEMBER,
    membershipRevision: "previous-membership",
  };
  const oldKey = issueDetailKeys.byIdentifier(oldMembershipScope, "PRJ", "7");
  const currentKey = issueDetailKeys.byIdentifier(scope, "PRJ", "7");
  client.setQueryData(oldKey, issue);
  client.setQueryData(currentKey, issue);
  setItemAccessStatus(client, scope, issue.id);
  const owner = captureIssueDetailReconciliation(client, scope, [issue.id]);
  assert.equal(await owner.reconcile({ ...issue, name: "Current membership receipt" }), 1);
  assert.equal(client.getQueryData(currentKey).name, "Current membership receipt");
  assert.equal(client.getQueryData(oldKey).name, "Old membership record");
});

await test("an accepted detail receipt cancels an older exact read before publishing", async (t) => {
  const { client, scope } = setup(t);
  const original = {
    id: "issue",
    project_id: scope.projectId,
    sequence_id: 7,
    name: "Original title",
  };
  const queryKey = issueDetailKeys.byIdentifier(scope, "PRJ", "7");
  setItemAccessStatus(client, scope, original.id);
  const pending = deferred();
  globalThis.detailTestFixture = {
    retrieveByIdentifier: () => {
      globalThis.detailTestReadStarted = true;
      return pending.promise;
    },
  };
  const read = client.fetchQuery({
    ...issueByIdentifierOptions(client, scope, "PRJ", "7"),
    staleTime: 0,
  });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(globalThis.detailTestReadStarted, true);
  client.setQueryData(queryKey, original);
  const owner = captureIssueDetailReconciliation(client, scope, [original.id]);
  const applied = await owner.reconcile({ ...original, name: "Saved title" });
  pending.resolve({ ...original, name: "Stale GET title" });
  await read.catch(() => undefined);
  assert.equal(applied, 1);
  assert.equal(client.getQueryData(queryKey).name, "Saved title");
  delete globalThis.detailTestReadStarted;
});

await test("a repeated sub-item response is normalized from canonical server records", () => {
  assert.deepEqual(allSubItems({ sub_issues: { open: [{ id: "a" }], done: [{ id: "b" }] } }), [
    { id: "a" },
    { id: "b" },
  ]);
});

await test("identifier reads accept the backend detail contract without a project identifier field", async (t) => {
  const { client, scope } = setup(t);
  const record = { id: "issue", project_id: scope.projectId, sequence_id: 7, name: "Canonical issue" };
  globalThis.detailTestFixture = { retrieveByIdentifier: async () => record };
  assert.deepEqual(await client.fetchQuery(issueByIdentifierOptions(client, scope, "PRJ", "7")), record);
});

await test("sub-item hook exposes grouped backend responses as rows with a successful read state", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { client, scope } = setup(t);
  installDetailHookScope(t, environment, scope);
  const child = { id: "child", project_id: scope.projectId, parent_id: "parent" };
  client.setQueryData(issueDetailKeys.subItems(scope, "parent"), { sub_issues: { started: [child] } });
  setItemAccessStatus(client, scope, "parent");
  function Probe() {
    const result = useWorkItemSubItems(scope.workspaceSlug, scope.projectId, "parent", scope.serviceType);
    return React.createElement("output", null, JSON.stringify({ rows: result.subItems, success: result.isSuccess }));
  }
  await environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(Probe)));
  assert.deepEqual(JSON.parse(environment.container.querySelector("output").textContent), {
    rows: [child],
    success: true,
  });
});

await test("an accepted no-content PATCH refreshes only its captured detail query", async (t) => {
  const { client, scope } = setup(t);
  const key = issueDetailKeys.detail(scope, "issue");
  client.setQueryData(key, { id: "issue", project_id: scope.projectId, name: "Before" });
  setItemAccessStatus(client, scope, "issue");
  const owner = captureIssueDetailReconciliation(client, scope, ["issue"]);

  assert.equal(await owner.reconcile(undefined), 0);
  assert.equal(client.getQueryState(key).isInvalidated, true);
  assert.equal(client.getQueryData(key).name, "Before");
});

await test("an accepted no-content PATCH cancels a pending initial detail read before refreshing", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { client, scope } = setup(t);
  installDetailHookScope(t, environment, scope);
  setItemAccessStatus(client, scope, "issue");

  const staleRead = deferred();
  const refreshedRead = deferred();
  let readCount = 0;
  let initialReadSignal;
  globalThis.detailTestFixture = {
    retrieve: (...args) => {
      readCount++;
      if (readCount === 1) {
        initialReadSignal = args.at(-1)?.signal;
        return staleRead.promise;
      }
      return refreshedRead.promise;
    },
  };

  await environment.render(
    React.createElement(QueryClientProvider, { client }, React.createElement(InitialDetailProbe, { client, scope }))
  );
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(readCount, 1, "the mounted detail observer started its initial read");

  const owner = captureIssueDetailReconciliation(client, scope, ["issue"]);
  let reconciliation;
  await React.act(async () => {
    reconciliation = owner.reconcile(undefined);
    await new Promise((resolve) => setImmediate(resolve));
  });
  assert.equal(initialReadSignal.aborted, true, "the pre-write GET was canceled");
  assert.equal(readCount, 2, "invalidation started a fresh GET for the active observer");

  await React.act(async () => {
    staleRead.resolve({ id: "issue", project_id: scope.projectId, name: "Stale pre-write title" });
    await new Promise((resolve) => setImmediate(resolve));
  });
  assert.equal(client.getQueryData(issueDetailKeys.detail(scope, "issue")), undefined);

  await React.act(async () => {
    refreshedRead.resolve({ id: "issue", project_id: scope.projectId, name: "Fresh post-write title" });
    await reconciliation;
    await new Promise((resolve) => setImmediate(resolve));
  });
  assert.equal(client.getQueryData(issueDetailKeys.detail(scope, "issue")).name, "Fresh post-write title");
});

await test("a replacement detail query is not invalidated while a no-content refresh cancels its predecessor", async (t) => {
  const { client, scope } = setup(t);
  const queryKey = issueDetailKeys.detail(scope, "issue");
  client.setQueryData(queryKey, { id: "issue", project_id: scope.projectId, name: "Original" });
  setItemAccessStatus(client, scope, "issue");
  const owner = captureIssueDetailReconciliation(client, scope, ["issue"]);
  const cancellationStarted = deferred();
  const finishCancellation = deferred();
  const cancelQueries = client.cancelQueries.bind(client);
  t.mock.method(client, "cancelQueries", async (...args) => {
    const result = await cancelQueries(...args);
    cancellationStarted.resolve();
    await finishCancellation.promise;
    return result;
  });

  const reconciliation = owner.reconcile(undefined);
  await cancellationStarted.promise;
  client.removeQueries({ queryKey, exact: true });
  const replacement = { id: "issue", project_id: scope.projectId, name: "Replacement owner" };
  client.setQueryData(queryKey, replacement);
  const replacementQuery = client.getQueryCache().find({ queryKey, exact: true });
  finishCancellation.resolve();
  await reconciliation;

  assert.equal(client.getQueryCache().find({ queryKey, exact: true }), replacementQuery);
  assert.equal(replacementQuery.state.isInvalidated, false);
  assert.deepEqual(client.getQueryData(queryKey), replacement);
});
