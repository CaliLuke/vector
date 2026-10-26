import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";

registerWebModuleHooks();
const source = "apps/web/core/hooks/use-workspace-invitation.tsx";
const formObserver = observeNativeForms([source]);
const { useInvitationForm } = await import("../../apps/web/core/hooks/use-workspace-invitation.tsx");

await test("invitation submission rejection is a native failure and retries the preserved rows", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const attempts = [];
  let shouldFail = true;
  t.after(() => environment.unmount());

  function Harness() {
    const form = useInvitationForm({
      onSubmit: async (data) => {
        attempts.push(data);
        if (shouldFail) throw new Error("offline");
      },
    });
    return React.createElement(
      "form",
      {
        onSubmit: (event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch((error) => form.setErrorMap({ onSubmit: error.message }));
        },
      },
      React.createElement(form.Field, { name: "emails[0].email" }, (field) =>
        React.createElement("input", {
          name: field.name,
          type: "email",
          value: field.state.value,
          onChange: (event) => field.handleChange(event.target.value),
        })
      ),
      React.createElement("button", { type: "submit" }, "Invite"),
      React.createElement(form.Subscribe, { selector: (state) => state.errorMap.onSubmit }, (error) =>
        React.createElement("p", { role: "alert" }, error ?? "")
      )
    );
  }

  await environment.render(React.createElement(Harness));
  const input = environment.container.querySelector('input[type="email"]');
  await changeFormControl(input, "person@example.com");
  await submitForm(environment.container);

  const formApi = formObserver.getForms(source).at(-1);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(input.value, "person@example.com");
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /offline/);
  assert.equal(attempts.length, 1);

  shouldFail = false;
  await submitForm(environment.container);
  assert.equal(formApi.state.isSubmitSuccessful, true);
  assert.equal(input.value, "");
  assert.equal(attempts.length, 2);
});

formObserver.restore();
