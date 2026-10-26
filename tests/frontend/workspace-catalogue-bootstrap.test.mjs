import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl, submitForm, waitForSubmitCompletion } from "./helpers/native-forms.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";

registerWebModuleHooks();
const stubs = {
  "@plane/i18n": `export const useTranslation = () => ({ t: (key) => key }); export const FALLBACK_LANGUAGE = "en";`,
};
globalThis.workspaceBootstrapReact = React;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/ui" && context.parentURL?.endsWith("/create-workspace-form.tsx"))
      return createModuleStub(`const React = globalThis.workspaceBootstrapReact;
        export const Input = ({hasError, ...props}) => React.createElement('input', props);
        export const FormErrors = () => null;
        const CustomSelect = Object.assign(
          ({value, onChange, disabled, children}) => React.createElement('select',
            {value, disabled, onChange:event => onChange(event.target.value)}, children),
          {Option: ({value, children}) => React.createElement('option', {value}, children)}
        );
        export {CustomSelect};`);
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

const formSource = "apps/web/core/components/workspace/invitation-selection-form.tsx";
const workspaceFormSource = "apps/web/core/components/workspace/create-workspace-form.tsx";
const formObserver = observeNativeForms([formSource, workspaceFormSource], { "@tanstack/react-form": ["useForm"] });

function renderJoinInvitesStep(render, Step, handleNextStep) {
  return render(
    React.createElement(Step, {
      invitations: [{ id: "invite", role: 20, workspace: { id: "workspace", name: "Team" } }],
      handleNextStep,
      handleCurrentViewChange: () => {},
    })
  );
}

await test("joining refresh failure keeps the continuation retryable without joining twice", async (t) => {
  const { environment, client, render } = await createQueryRouterFixture(t, "/acme/");
  const user = { id: "user" };
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const { UserService } = await import("../../apps/web/core/services/user.service.ts");
  const { WorkspaceJoinInvitesStep } =
    await import("../../apps/web/core/components/onboarding/steps/workspace/join-invites.tsx");
  client.setQueryData(identityKeys.currentUser(), user);
  let failCatalogueRefresh = true;
  let joins = 0;
  let catalogueReads = 0;
  let settingsRefreshes = 0;
  let continued = 0;
  t.mock.method(WorkspaceService.prototype, "joinWorkspaces", async () => {
    joins++;
  });
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    catalogueReads++;
    if (failCatalogueRefresh) throw new Error("offline");
    return [{ id: "workspace", slug: "team" }];
  });
  t.mock.method(UserService.prototype, "currentUserSettings", async () => {
    settingsRefreshes++;
    return { id: user.id, workspace: {} };
  });
  await renderJoinInvitesStep(render, WorkspaceJoinInvitesStep, async () => {
    continued++;
  });
  const checkbox = environment.container.querySelector('input[type="checkbox"]');
  await act(async () => checkbox.click());
  await submitForm(environment.container);

  assert.equal(joins, 1);
  assert.equal(catalogueReads, 1);
  assert.equal(continued, 0);
  assert.match(environment.container.querySelector('[role="alert"]')?.textContent ?? "", /Could not finish joining/);
  assert.equal(formObserver.getForms(formSource).at(-1)?.state.isSubmitSuccessful, false);

  failCatalogueRefresh = false;
  await submitForm(environment.container);

  assert.equal(joins, 1);
  assert.equal(catalogueReads, 2);
  assert.equal(settingsRefreshes, 1);
  assert.equal(continued, 1);
  assert.equal(formObserver.getForms(formSource).at(-1)?.state.isSubmitSuccessful, true);
});

await test("late invitation acceptance cannot continue after the identity generation changes", async (t) => {
  const { environment, client, render } = await createQueryRouterFixture(t, "/acme/");
  const user = { id: "user" };
  const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const { WorkspaceJoinInvitesStep } =
    await import("../../apps/web/core/components/onboarding/steps/workspace/join-invites.tsx");
  client.setQueryData(identityKeys.currentUser(), user);
  let resolveJoin;
  const joinPending = new Promise((resolve) => {
    resolveJoin = resolve;
  });
  let joinCalls = 0;
  let catalogueReads = 0;
  let continued = 0;
  t.mock.method(WorkspaceService.prototype, "joinWorkspaces", async () => {
    joinCalls++;
    return joinPending;
  });
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    catalogueReads++;
    return [{ id: "workspace", slug: "team" }];
  });
  await renderJoinInvitesStep(render, WorkspaceJoinInvitesStep, async () => {
    continued++;
  });
  await act(async () => environment.container.querySelector('input[type="checkbox"]').click());
  await submitForm(environment.container);
  assert.equal(joinCalls, 1);

  advanceIdentitySession(client);
  resolveJoin();
  await act(() => waitForSubmitCompletion(formObserver.getForms(formSource).at(-1)));

  assert.equal(catalogueReads, 0);
  assert.equal(continued, 0);
  assert.equal(formObserver.getForms(formSource).at(-1)?.state.isSubmitSuccessful, false);
  const submit = environment.container.querySelector('button[type="submit"]');
  assert.equal(submit.disabled, false);
  await act(async () => submit.click());
  await act(() => waitForSubmitCompletion(formObserver.getForms(formSource).at(-1)));
  assert.equal(joinCalls, 1, "retrying the stale continuation must not repeat the accepted join");
  assert.equal(catalogueReads, 0);
  assert.equal(continued, 0);
});

await test("workspace creation stops before settings refresh when the session changes during profile save", async (t) => {
  const { environment, client, render } = await createQueryRouterFixture(t, "/acme/");
  const user = { id: "user" };
  const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const { UserService } = await import("../../apps/web/core/services/user.service.ts");
  const { WorkspaceCreateStep } = await import("../../apps/web/core/components/onboarding/steps/workspace/create.tsx");
  client.setQueryData(identityKeys.currentUser(), user);
  let resolveProfileSave;
  const profileSave = new Promise((resolve) => {
    resolveProfileSave = resolve;
  });
  let markProfileSaveStarted;
  const profileSaveStarted = new Promise((resolve) => {
    markProfileSaveStarted = resolve;
  });
  let creates = 0;
  let profileWrites = 0;
  let settingsReads = 0;
  let continued = 0;
  t.mock.method(WorkspaceService.prototype, "workspaceSlugCheck", async () => ({ status: true }));
  t.mock.method(WorkspaceService.prototype, "createWorkspace", async (data) => {
    creates++;
    return { id: "workspace", ...data };
  });
  t.mock.method(UserService.prototype, "updateCurrentUserProfile", async () => {
    profileWrites++;
    markProfileSaveStarted();
    return profileSave;
  });
  t.mock.method(UserService.prototype, "currentUserSettings", async () => {
    settingsReads++;
    return { id: user.id, workspace: {} };
  });
  await render(
    React.createElement(WorkspaceCreateStep, {
      user,
      onComplete: async () => {
        continued++;
      },
      handleCurrentViewChange: () => {},
    })
  );
  const named = environment.container.querySelector('[name="name"]');
  const slug = environment.container.querySelector('[name="slug"]');
  assert.ok(named && slug, environment.container.innerHTML);
  await changeFormControl(named, "Team");
  await changeFormControl(slug, "team");
  await changeFormControl(environment.container.querySelector("select"), "Just myself");
  await submitForm(environment.container);
  await profileSaveStarted;

  advanceIdentitySession(client);
  resolveProfileSave({ user: user.id, onboarding_step: {}, last_workspace_id: "workspace" });
  await act(() => waitForSubmitCompletion(formObserver.getForms(workspaceFormSource).at(-1)));

  assert.equal(creates, 1);
  assert.equal(profileWrites, 1);
  assert.equal(settingsReads, 0);
  assert.equal(continued, 0);
  assert.equal(formObserver.getForms(workspaceFormSource).at(-1)?.state.isSubmitSuccessful, false);
  const submit = environment.container.querySelector('button[type="submit"]');
  assert.equal(submit.disabled, false);
  await act(async () => submit.click());
  await act(() => waitForSubmitCompletion(formObserver.getForms(workspaceFormSource).at(-1)));
  assert.equal(creates, 1, "retrying the post-create continuation must reuse the created workspace");
  assert.equal(profileWrites, 1, "a stale session must not begin another profile update");
  assert.equal(settingsReads, 0);
  assert.equal(continued, 0);
});

await test("a workspace create without a loaded catalogue waits for the complete server list", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const user = { id: "user" };
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { createWorkspaceOptions, workspaceKeys, workspacesOptions } =
    await import("../../apps/web/core/queries/workspaces.ts");
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  client.setQueryData(identityKeys.currentUser(), user);
  const created = { id: "created", slug: "team", name: "Team" };
  const canonical = { ...created, member_role: 20, logo_url: null };
  t.mock.method(WorkspaceService.prototype, "createWorkspace", async () => created);
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => [canonical]);
  t.after(() => client.clear());

  const mutation = client.getMutationCache().build(client, createWorkspaceOptions(client, user.id));
  assert.deepEqual(await mutation.execute({ name: "Team", slug: "team" }), created);
  assert.equal(client.getQueryData(workspaceKeys.list(user.id)), undefined);

  const completeList = await client.fetchQuery(workspacesOptions(user.id));
  assert.deepEqual(completeList, [canonical]);
  assert.deepEqual(client.getQueryData(workspaceKeys.list(user.id)), [canonical]);
});

formObserver.restore();
