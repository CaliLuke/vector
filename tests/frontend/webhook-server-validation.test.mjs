import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";

registerWebModuleHooks();
const source = "apps/web/core/components/web-hooks/form/form.tsx";
const formObserver = observeNativeForms([source]);

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const reactURL = pathToFileURL(webRequire.resolve("react")).href;
const substitutes = {
  "@plane/constants": "export const WORKSPACE_SETTINGS_TRACKER_ELEMENTS = { WEBHOOK_UPDATE_BUTTON: 'webhook-update' };",
  "@plane/i18n": "export const useTranslation = () => ({ t: (key) => key });",
  "@plane/propel/button":
    "import React from 'react'; export const Button = ({ loading, ...props }) => React.createElement('button', props, props.children);",
  "@plane/types": "",
  "./secret-key": "export const WebhookSecretKey = () => null;",
  "@plane/ui": `import React from "react";
    export { Input } from ${JSON.stringify(pathToFileURL(path.join(repositoryRoot, "packages/ui/src/form-fields/input.tsx")).href)};
    export const Checkbox = props => React.createElement("input", {...props, type: "checkbox"});
    export const ToggleSwitch = ({value, onChange, label, disabled}) => React.createElement("input", {type: "checkbox", "aria-label":label, checked:value, disabled, onChange: event => onChange(event.target.checked)});`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "react") return { shortCircuit: true, url: reactURL };
    if (specifier === "../utils" && context.parentURL?.endsWith("/form-fields/input.tsx")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');")}`,
      };
    }
    if (Object.hasOwn(substitutes, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(substitutes[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

await test("WebhookForm displays a server URL validation error beside the field", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { WebhookForm } = await import("../../apps/web/core/components/web-hooks/form/form.tsx");

  await environment.render(
    React.createElement(WebhookForm, {
      data: { url: "https://retirement-smoke.invalid/hook" },
      currentWorkspace: null,
      onSubmit: async () => {
        throw { url: ["The destination host could not be resolved."] };
      },
    })
  );

  const input = environment.container.querySelector("input");
  assert.ok(input);
  await act(async () => environment.container.querySelector('button[type="submit"]').click());

  assert.equal(
    environment.container.querySelector('[role="alert"]')?.textContent,
    "The destination host could not be resolved."
  );
  assert.equal(input.getAttribute("aria-invalid"), "true");
});

await test("switching a webhook from individual events to all submits every event and retains failures for retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { WebhookForm } = await import("../../apps/web/core/components/web-hooks/form/form.tsx");
  let fail = true;
  const payloads = [];
  await environment.render(
    React.createElement(WebhookForm, {
      data: {
        url: "https://example.com/hook",
        project: true,
        cycle: false,
        module: false,
        issue: false,
        issue_comment: false,
        is_active: false,
      },
      currentWorkspace: null,
      onSubmit: async (value) => {
        payloads.push(value);
        if (fail) throw new Error("offline");
      },
    })
  );
  await act(async () => environment.container.querySelector('input[type="radio"][value="all"]').click());
  const submit = () => act(async () => environment.container.querySelector('button[type="submit"]').click());
  await submit();
  assert.ok(environment.container.querySelector('[role="alert"]'));
  const formApi = formObserver.getForms(source).at(-1);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  fail = false;
  await submit();
  assert.equal(formApi.state.isSubmitSuccessful, true);
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
  assert.equal(payloads.length, 2);
  assert.deepEqual(payloads[1], {
    url: "https://example.com/hook",
    is_active: false,
    project: true,
    cycle: true,
    module: true,
    issue: true,
    issue_comment: true,
  });
});

formObserver.restore();
