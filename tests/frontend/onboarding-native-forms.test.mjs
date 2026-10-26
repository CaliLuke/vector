import { seedInstanceInfo } from "./helpers/instance-info.mjs";
import { changeFormControl as change, submitForm as submit } from "./helpers/native-forms.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { MutationObserver } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
registerWebModuleHooks();
const workspaceFormSource = "apps/web/core/components/workspace/create-workspace-form.tsx";
const profileFormSource = "apps/web/core/components/onboarding/steps/profile/root.tsx";
const formObserver = observeNativeForms([workspaceFormSource, profileFormSource]);
globalThis.onboardingFormsReact = React;
const stubs = {
  "@/components/core/modals/user-image-upload-modal": `export const UserImageUploadModal = () => null;`,
  "@plane/i18n": `export const FALLBACK_LANGUAGE = "en"; export const useTranslation = () => ({t:key => key}); export const setLanguage = async () => {};`,
  "@plane/propel/toast": `export const TOAST_TYPE = {ERROR:"error", SUCCESS:"success"}; export const setToast = () => {};`,
  "@plane/propel/button": `const React = globalThis.onboardingFormsReact; export const Button = ({loading, variant, size, children, ...props}) => React.createElement("button", props, children);`,
  "@plane/ui": `export { Form, FormErrors } from "${new URL("../../packages/ui/src/form-fields/root.tsx", import.meta.url).href}";
    export { ModalForm } from "${new URL("../../packages/ui/src/modals/modal-core.tsx", import.meta.url).href}";
    const React = globalThis.onboardingFormsReact;
    export const Input = ({hasError, ...props}) => React.createElement("input", props);
    export const CustomSelect = Object.assign(({children, value, onChange, disabled}) => React.createElement("select", {value, disabled, onChange:event=>onChange(event.target.value)}, React.createElement("option", {value:""}, "Select"), children), {Option: ({children, value}) => React.createElement("option", {value}, children)});
    export const AuthPasswordInput = ({label, ...props}) => React.createElement("input", props);
    export const AuthConfirmPasswordInput = ({label, password, ...props}) => React.createElement("input", props);`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const settle = () => act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
async function fixture(t) {
  const scope = await createQueryRouterFixture(t, "/create-workspace");
  seedInstanceInfo(scope.client, { is_self_managed: false });
  return scope;
}

await test("workspace creation preserves edited slugs, retries checks and continues without creating twice", async (t) => {
  const { environment, render, client } = await fixture(t);
  const { CreateWorkspaceForm } = await import("../../apps/web/core/components/workspace/create-workspace-form.tsx");
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  client.setQueryData(currentUserOptions().queryKey, { id: "user" });
  const creates = [];
  let available = false,
    failCompletion = true,
    completions = 0;
  t.mock.method(WorkspaceService.prototype, "createWorkspace", async (data) => {
    creates.push(data);
    return { id: "workspace", ...data };
  });
  t.mock.method(WorkspaceService.prototype, "workspaceSlugCheck", async () => ({ status: available }));
  await render(
    React.createElement(CreateWorkspaceForm, {
      onSubmit: async () => {
        completions++;
        if (failCompletion) throw new Error("offline");
      },
    })
  );
  const input = (name) => environment.container.querySelector(`[name="${name}"]`);
  await change(input("name"), "Team");
  assert.equal(input("slug").value, "team");
  await change(input("slug"), "custom");
  await change(input("name"), "Updated team");
  assert.equal(input("slug").value, "custom");
  const select = environment.container.querySelector("select");
  await change(select, select.options[1].value);
  await submit(environment.container);
  await settle();
  assert.equal(creates.length, 0);
  assert.match(environment.container.textContent, /url_already_taken/);
  assert.equal(environment.container.querySelector('button[type="submit"]').disabled, false);
  available = true;
  await submit(environment.container);
  await settle();
  assert.equal(creates.length, 1);
  assert.equal(completions, 1);
  assert.equal(input("slug").value, "custom");
  assert.equal(environment.container.querySelector("fieldset").disabled, true);
  assert.equal(formObserver.getForms(workspaceFormSource).at(-1)?.state.isSubmitSuccessful, false);
  assert.equal(environment.container.querySelector('button[type="submit"]').disabled, false);
  failCompletion = false;
  await submit(environment.container);
  await settle();
  assert.equal(creates.length, 1);
  assert.equal(completions, 2);
});
await test("profile setup retains edits on failure and advances only after all updates succeed", async (t) => {
  const { environment, render, client } = await fixture(t);
  const { ProfileSetupStep } = await import("../../apps/web/core/components/onboarding/steps/profile/root.tsx");
  const { UserService } = await import("../../apps/web/core/services/user.service.ts");
  const { currentUserOptions, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const user = { id: "user", first_name: "Initial", last_name: "Name", is_password_autoset: false };
  const savedProfile = {
    id: "profile",
    user: user.id,
    role: undefined,
    last_workspace_id: undefined,
    theme: { theme: "system", primary: undefined, background: undefined, darkPalette: false },
    onboarding_step: {
      profile_complete: false,
      workspace_join: false,
      workspace_create: false,
      workspace_invite: false,
    },
    is_onboarded: false,
    is_tour_completed: false,
    use_case: undefined,
    billing_address_country: undefined,
    billing_address: undefined,
    has_billing_address: false,
    has_marketing_email_consent: false,
    created_at: "",
    updated_at: "",
    language: "en",
    start_of_the_week: 0,
  };
  client.setQueryData(currentUserOptions().queryKey, user);
  client.setQueryData(identityKeys.profile(user.id), savedProfile);
  let fail = true,
    advanced = 0;
  const names = [],
    consents = [];
  t.mock.method(UserService.prototype, "updateUser", async (value) => {
    names.push(value);
    return { ...user, ...value };
  });
  t.mock.method(UserService.prototype, "updateCurrentUserProfile", async (value) => {
    consents.push(value);
    if (fail) throw new Error("offline");
    Object.assign(savedProfile, value);
    return { ...savedProfile };
  });
  await render(React.createElement(ProfileSetupStep, { handleStepChange: async () => advanced++ }));
  await change(environment.container.querySelector('[name="first_name"]'), "Changed");
  await submit(environment.container);
  assert.equal(advanced, 0);
  assert.equal(client.getQueryData(identityKeys.profile(user.id)).has_marketing_email_consent, false);
  assert.equal(environment.container.querySelector('[name="first_name"]').value, "Changed");
  assert.equal(formObserver.getForms(profileFormSource).at(-1)?.state.isSubmitSuccessful, false);
  assert.equal(environment.container.querySelector('button[type="submit"]').disabled, false);
  fail = false;
  await submit(environment.container);
  assert.equal(advanced, 1);
  assert.equal(client.getQueryData(identityKeys.profile(user.id)).has_marketing_email_consent, true);
  assert.equal(names.length, 2);
  assert.equal(consents.length, 2);
});
await test("onboarding skip waits for completion and exposes retry after failure", async (t) => {
  const { environment, render } = await fixture(t);
  const { OnboardingActions } = await import("../../apps/web/core/components/onboarding/steps/common/actions.tsx");
  let finish;
  await render(
    React.createElement(OnboardingActions, {
      canSubmit: true,
      isSubmitting: false,
      onSkip: () =>
        new Promise((resolve, reject) => {
          finish = { resolve, reject };
        }),
    })
  );
  const buttons = environment.container.querySelectorAll("button");
  await act(async () => buttons[1].click());
  await settle();
  assert.equal(buttons[0].disabled, true);
  assert.equal(buttons[1].disabled, true);
  await act(async () => finish.reject(new Error("offline")));
  await settle();
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Unable to continue/);
  await act(async () => buttons[1].click());
  await settle();
  await act(async () => finish.resolve());
  await settle();
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
});
await test("email notification mutations retain saved values on failure and update the shared cache on success", async (t) => {
  const { client } = await fixture(t);
  const { UserService } = await import("../../apps/web/core/services/user.service.ts");
  const { currentUserOptions, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { emailNotificationSettingsOptions, emailNotificationSettingsMutationOptions } =
    await import("../../apps/web/core/queries/user-settings.ts");
  const user = { id: "user" };
  client.setQueryData(currentUserOptions().queryKey, user);
  const saved = { property_change: false, state_change: true, issue_completed: true, comment: true, mention: true };
  t.mock.method(UserService.prototype, "currentUserEmailNotificationSettings", async () => ({ ...saved }));
  let fail = true;
  t.mock.method(UserService.prototype, "updateCurrentUserEmailNotificationSettings", async (patch) => {
    if (fail) throw new Error("offline");
    Object.assign(saved, patch);
    return { ...saved };
  });
  await client.fetchQuery(emailNotificationSettingsOptions(user.id));
  const mutate = (value) =>
    client.getMutationCache().build(client, emailNotificationSettingsMutationOptions(client, user.id)).execute(value);
  await assert.rejects(mutate({ property_change: true }), /offline/);
  assert.equal(client.getQueryData(identityKeys.emailNotificationSettings(user.id)).property_change, false);
  fail = false;
  await mutate({ property_change: true });
  assert.equal(client.getQueryData(identityKeys.emailNotificationSettings(user.id)).property_change, true);
});

await test("serialized native profile mutations preserve a concurrent successful preference", async (t) => {
  const { client } = await fixture(t);
  const { UserService } = await import("../../apps/web/core/services/user.service.ts");
  const { currentUserOptions, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { updateUserProfileOptions } = await import("../../apps/web/core/queries/user-profile.ts");
  const user = { id: "user" };
  const profile = { user: user.id, role: undefined, has_marketing_email_consent: false };
  client.setQueryData(currentUserOptions().queryKey, user);
  client.setQueryData(identityKeys.profile(user.id), profile);
  const pending = Promise.withResolvers();
  t.mock.method(UserService.prototype, "updateCurrentUserProfile", async (value) => {
    if ("has_marketing_email_consent" in value) return pending.promise;
    return { ...profile, ...value };
  });
  const options = updateUserProfileOptions(client, user.id);
  const failedMutation = client.getMutationCache().build(client, options);
  const nextMutation = client.getMutationCache().build(client, options);
  const failedSave = assert.rejects(failedMutation.execute({ has_marketing_email_consent: true }), /offline/);
  const nextSave = nextMutation.execute({ role: "developer" });
  await Promise.resolve();
  pending.reject(new Error("offline"));
  await failedSave;
  await nextSave;
  assert.deepEqual(client.getQueryData(identityKeys.profile(user.id)), {
    ...profile,
    role: "developer",
  });
});

await test("onboarding durable writes succeed natively when only follow-up settings refresh fails", async (t) => {
  const { client } = await fixture(t);
  const { UserService } = await import("../../apps/web/core/services/user.service.ts");
  const { currentUserOptions, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { finishUserOnboardingOptions } = await import("../../apps/web/core/queries/user-profile.ts");
  const user = { id: "user" };
  const profile = {
    id: "profile",
    user: user.id,
    onboarding_step: {
      profile_complete: false,
      workspace_join: false,
      workspace_create: false,
      workspace_invite: false,
    },
    is_onboarded: false,
    has_marketing_email_consent: false,
  };
  client.setQueryData(currentUserOptions().queryKey, user);
  client.setQueryData(identityKeys.profile(user.id), profile);
  const writes = [];
  t.mock.method(UserService.prototype, "updateCurrentUserProfile", async (profileChange) => {
    writes.push("profile");
    return { ...profile, ...profileChange };
  });
  t.mock.method(UserService.prototype, "updateUserOnBoard", async () => {
    writes.push("onboard");
    return { message: "Onboarding completed" };
  });
  t.mock.method(UserService.prototype, "getCurrentUserProfile", async () => ({
    ...profile,
    onboarding_step: { profile_complete: true, workspace_join: true, workspace_create: true, workspace_invite: true },
    last_workspace_id: "workspace",
    is_onboarded: true,
  }));
  t.mock.method(UserService.prototype, "currentUserSettings", async () => {
    throw new Error("settings refresh unavailable");
  });

  const mutation = new MutationObserver(client, finishUserOnboardingOptions(client, user.id));
  const result = await mutation.mutate("workspace");
  assert.deepEqual(writes, ["profile", "onboard"]);
  assert.equal(mutation.getCurrentResult().isSuccess, true);
  assert.equal(result.profile.is_onboarded, true);
  assert.match(result.refreshError.message, /settings refresh unavailable/);
  assert.equal(client.getQueryData(identityKeys.profile(user.id)).is_onboarded, true);
});

formObserver.restore();
