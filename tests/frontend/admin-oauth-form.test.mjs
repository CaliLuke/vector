import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import {
  setupAdminSettingsFormTest,
  formToasts as toasts,
  changeInput as change,
  submitAndWait,
} from "./helpers/admin-forms.mjs";
const configurationFormSource = "apps/admin/components/common/use-configuration-form.ts";
const formObserver = setupAdminSettingsFormTest([configurationFormSource], "./settings-form");

for (const provider of ["google", "github", "gitlab", "gitea"]) {
  void test(`${provider} saves only its own credentials and retains edits after failure`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const { MemoryRouter } = createRequire(new URL("../../apps/admin/package.json", import.meta.url))("react-router");
    const { OAuthConfigurationForm } =
      await import("../../apps/admin/components/authentication/oauth-configuration-form.tsx");
    const { oauthSettings } = await import("../../apps/admin/components/authentication/oauth-settings.tsx");
    const settings = oauthSettings[provider];
    const field = settings.fields.find((item) => item.key.endsWith("CLIENT_ID"));
    const config = { EMAIL_HOST_PASSWORD: "unrelated-private-value" };
    for (const item of settings.fields) config[item.key] = "original";
    let failed = true;
    const requests = [];
    await environment.render(
      React.createElement(
        MemoryRouter,
        null,
        React.createElement(OAuthConfigurationForm, {
          settings,
          config,
          save: async (payload) => {
            requests.push(payload);
            if (failed) throw new Error("offline");
            return Object.entries(payload).map(([key, value]) => ({ key, value }));
          },
        })
      )
    );
    const input = environment.container.querySelector(`input[name="${field.key}"]`);
    await change(input, "changed");
    const save = () =>
      [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "Save changes");
    const formApi = formObserver.getForms(configurationFormSource).at(-1);
    assert.equal(save().disabled, false);
    await submitAndWait(save(), formApi);
    assert.equal(formApi.state.isSubmitSuccessful, false);
    assert.equal(toasts.at(-1).type, "error");
    assert.equal(input.value, "changed");
    assert.equal(save().disabled, false);
    failed = false;
    await submitAndWait(save(), formApi);
    assert.equal(formApi.state.isSubmitSuccessful, true);
    assert.equal(toasts.at(-1).type, "success");
    assert.equal(save().disabled, true);
    assert.equal(requests[1][field.key], "changed");
    assert.equal(Object.hasOwn(requests[1], "EMAIL_HOST_PASSWORD"), false);
  });
}
