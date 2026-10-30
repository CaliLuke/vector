import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";

registerWebModuleHooks();
const nativeForms = observeNativeForms(["apps/web/core/components/issues/work-item-selection-modal.tsx"]);
const { EInboxIssueStatus } = webRequire("@plane/types");
const stubs = {
  "@plane/i18n": `export const useTranslation = () => ({t: (key) => key});`,
  "@/components/issues/issue-detail/issue-identifier": `export const IssueIdentifier = () => null;`,
  "@/hooks/store/use-project": `export const useProject = () => ({getProjectById: () => ({identifier: "TEST"})});`,
  "@/components/empty-state/simple-empty-state-root": `export const SimpleEmptyState = () => null;`,
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.endsWith(".webp?url")) return createModuleStub('export default "test-image.webp";');
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});
const { MutationObserver } = webRequire("@tanstack/react-query");
const { intakeMutationOptions, intakeKeys } = await import("../../apps/web/core/queries/intake.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { InboxIssueService } = await import("../../apps/web/core/services/inbox/inbox-issue.service.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const initialIssue = {
  id: "intake",
  status: EInboxIssueStatus.PENDING,
  issue: { id: "current", project_id: "project" },
};
function seedSearchPermissions(client, projectRoles) {
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  client.setQueryData(permissionKeys.workspaceMembership("alice", "workspace"), {
    is_active: true,
    role: 15,
    updated_at: "v1",
  });
  client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "workspace", 15, "v1"), projectRoles);
}
function intake(t) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
  });
  t.after(() => client.clear());
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
  seedSearchPermissions(client, { project: 20 });
  const key = intakeKeys.detail(scope, "current");
  client.setQueryData(key, initialIssue);
  const project = { refreshes: 0 };
  const mutation = new MutationObserver(
    client,
    intakeMutationOptions(client, scope, () => ({
      beforeWrite: async () => {},
      accepted: async () => {
        project.refreshes++;
      },
    }))
  );
  const detail = () => client.getQueryData(key);
  const mutate = (data) => mutation.mutate({ kind: "status", issueId: "current", data });
  return {
    client,
    mutate,
    detail,
    project,
    mutateIssue: (data) => mutation.mutate({ kind: "issue", issueId: "current", data }),
  };
}
await test("failed snooze and unsnooze requests preserve the saved status and do not refresh project counts", async (t) => {
  const { mutate, detail, project } = intake(t);
  const date = new Date("2030-01-01T00:00:00Z");
  t.mock.method(InboxIssueService.prototype, "update", async () => {
    throw new Error("offline");
  });
  await assert.rejects(mutate({ status: EInboxIssueStatus.SNOOZED, snoozed_till: date }), /offline/);
  assert.equal(detail().status, EInboxIssueStatus.PENDING);
  assert.equal(detail().snoozed_till, undefined);
  assert.equal(project.refreshes, 0);
  t.mock.method(InboxIssueService.prototype, "update", async () => ({
    ...initialIssue,
    status: EInboxIssueStatus.SNOOZED,
    snoozed_till: date,
  }));
  await mutate({ status: EInboxIssueStatus.SNOOZED, snoozed_till: date });
  t.mock.method(InboxIssueService.prototype, "update", async () => {
    throw new Error("offline");
  });
  await assert.rejects(mutate({ status: EInboxIssueStatus.PENDING, snoozed_till: null }), /offline/);
  assert.equal(detail().status, EInboxIssueStatus.SNOOZED);
  assert.equal(detail().snoozed_till.toISOString(), date.toISOString());
  assert.equal(project.refreshes, 1);
});
await test("duplicate selection waits for the native mutation request and retains the selection after failure", async (t) => {
  const env = await createReactDomTestEnvironment();
  const { client, mutate, detail, project } = intake(t);
  t.after(async () => env.unmount());
  let closed = 0,
    attempts = 0;
  const pending = Promise.withResolvers();
  const searches = [];
  t.mock.method(ProjectService.prototype, "projectIssuesSearch", async (...args) => {
    searches.push(args);
    return [
      { id: "current", name: "This intake item", project_id: "project", sequence_id: 1 },
      { id: "duplicate", name: "Existing work item", project_id: "project", sequence_id: 2 },
    ];
  });
  t.mock.method(InboxIssueService.prototype, "update", async () => {
    attempts++;
    if (attempts === 1) return pending.promise;
    return { ...initialIssue, status: EInboxIssueStatus.DUPLICATE, duplicate_to: "duplicate" };
  });
  const { WorkItemSelectionModal } =
    await import("../../apps/web/core/components/issues/work-item-selection-modal.tsx");
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(WorkItemSelectionModal, {
        isOpen: true,
        workspaceSlug: "workspace",
        projectId: "project",
        issueId: "current",
        onClose: () => closed++,
        onSelect: (issue) => mutate({ status: EInboxIssueStatus.DUPLICATE, duplicate_to: issue.id }),
      })
    )
  );
  await act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
  const options = document.querySelectorAll('[role="option"]');
  assert.equal(options.length, 1, "the intake item itself cannot be selected");
  assert.equal(searches[0][0], "workspace", "explicit scope works without a project route");
  assert.equal(searches[0][1], "project");
  assert.equal(searches[0][2].workspace_search, false, "duplicate selection stays in the intake project");
  await act(async () => {
    options[0].dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true }));
    options[0].click();
  });
  assert.equal(document.querySelector("fieldset").disabled, true);
  assert.equal(closed, 0);
  await act(async () => pending.reject(new Error("offline")));
  assert.match(document.querySelector('[role="alert"]').textContent, /Could not save/);
  assert.equal(detail().status, EInboxIssueStatus.PENDING);
  assert.equal(project.refreshes, 0);
  assert.equal(document.querySelector('[role="option"]').getAttribute("aria-selected"), "true");
  assert.equal(
    nativeForms.getForms("apps/web/core/components/issues/work-item-selection-modal.tsx").at(-1).state
      .isSubmitSuccessful,
    false
  );
  assert.equal(document.querySelector('button[type="submit"]').disabled, false);
  await act(async () => document.querySelector('button[type="submit"]').click());
  assert.equal(closed, 1);
  assert.equal(
    nativeForms.getForms("apps/web/core/components/issues/work-item-selection-modal.tsx").at(-1).state
      .isSubmitSuccessful,
    true
  );
  assert.equal(detail().duplicate_to, "duplicate");
  assert.equal(project.refreshes, 1);
});
await test("changing the selector scope cancels its old search and excludes late results", async (t) => {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  const pending = Promise.withResolvers();
  seedSearchPermissions(client, { first: 20, second: 20 });
  let firstSignal;
  t.mock.method(ProjectService.prototype, "projectIssuesSearch", async (_workspace, project, params, signal) => {
    assert.equal(params.workspace_search, true, "parents are eligible across readable workspace projects");
    assert.equal(params.child_type_id, "selected-type");
    if (project === "first") {
      firstSignal = signal;
      return pending.promise;
    }
    return [
      {
        id: "second-item",
        name: "Second project result",
        project_id: project,
        project__identifier: "SECOND",
        sequence_id: 2,
      },
    ];
  });
  const { WorkItemSelectionModal } =
    await import("../../apps/web/core/components/issues/work-item-selection-modal.tsx");
  const render = (projectId) =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(WorkItemSelectionModal, {
          isOpen: true,
          workspaceSlug: "workspace",
          projectId,
          onClose: () => {},
          onSelect: () => {},
          filters: { parent: true, child_type_id: "selected-type" },
        })
      )
    );
  await render("first");
  assert.equal(firstSignal.aborted, false);
  await render("second");
  await act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
  assert.equal(firstSignal.aborted, true);
  assert.match(document.querySelector('[role="option"]').textContent, /Second project result/);
  await act(async () =>
    pending.resolve([{ id: "old-item", name: "Old project result", project_id: "first", sequence_id: 1 }])
  );
  assert.match(document.querySelector('[role="option"]').textContent, /Second project result/);
  assert.equal(document.querySelectorAll('[role="option"]').length, 1);
});

nativeForms.restore();

await test("failed intake text writes retain the saved record and commit a successful retry", async (t) => {
  const { mutateIssue, detail } = intake(t);
  const request = Promise.withResolvers();
  t.mock.method(InboxIssueService.prototype, "updateIssue", async () => request.promise);
  const write = mutateIssue({ name: "Draft title" });
  const rejected = assert.rejects(write, /offline/);
  assert.equal(detail().issue.name, undefined);
  request.reject(new Error("offline"));
  await rejected;
  assert.equal(detail().issue.name, undefined);
  t.mock.method(InboxIssueService.prototype, "updateIssue", async () => ({
    ...initialIssue,
    issue: { ...initialIssue.issue, name: "Retried title" },
  }));
  await mutateIssue({ name: "Retried title" });
  assert.equal(detail().issue.name, "Retried title");
});
