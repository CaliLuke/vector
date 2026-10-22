import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { React, act, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const reactURL = pathToFileURL(webRequire.resolve("react")).href;
const webhookInputURL = pathToFileURL(
  path.join(repositoryRoot, "apps/web/core/components/web-hooks/form/input.tsx")
).href;
const substitutes = {
  "@plane/constants": "export const WORKSPACE_SETTINGS_TRACKER_ELEMENTS = { WEBHOOK_UPDATE_BUTTON: 'webhook-update' };",
  "@plane/i18n": "export const useTranslation = () => ({ t: (key) => key });",
  "@plane/propel/button":
    "import React from 'react'; export const Button = ({ loading, ...props }) => React.createElement('button', props, props.children);",
  "@plane/types": "",
  "mobx-react": "export const observer = (component) => component;",
  "@/hooks/store/use-webhook": "export const useWebhook = () => ({ webhookSecretKey: null });",
  "@/components/web-hooks": `export { WebhookInput } from ${JSON.stringify(webhookInputURL)}; export const WebhookIndividualEventOptions = () => null; export const WebhookOptions = () => null; export const WebhookSecretKey = () => null; export const WebhookToggle = () => null;`,
  "@plane/ui": `export { Input } from ${JSON.stringify(pathToFileURL(path.join(repositoryRoot, "packages/ui/src/form-fields/input.tsx")).href)};`,
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

test("WebhookForm displays a server URL validation error beside the field", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { WebhookForm } = await import("../../apps/web/core/components/web-hooks/form/form.tsx");

  await environment.render(
    React.createElement(WebhookForm, {
      data: { url: "https://retirement-smoke.invalid/hook" },
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
