import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const testModules = {
  "@plane/ui": "tests/frontend/helpers/project-state-form-ui.mjs",
  "@plane/propel/button": "tests/frontend/helpers/project-state-form-button.mjs",
  "react-color": "tests/frontend/helpers/project-state-form-color-picker.mjs",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    const relativePath = testModules[specifier];
    if (relativePath) {
      return nextResolve(pathToFileURL(path.join(repositoryRoot, relativePath)).href, context);
    }
    return nextResolve(specifier, context);
  },
});

test("StateForm keeps its name input controlled on the first render", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  const warnings = [];
  const originalError = console.error;
  console.error = (...args) => warnings.push(args.join(" "));
  t.after(() => {
    console.error = originalError;
  });

  const { StateForm } = await import("../../apps/web/core/components/project-states/create-update/form.tsx");
  await environment.render(
    React.createElement(StateForm, {
      data: { name: "Review", description: "", color: "#778899" },
      onSubmit: async () => ({ status: "success" }),
      onCancel: () => {},
      buttonDisabled: false,
      buttonTitle: "Save",
    })
  );

  assert.equal(environment.container.querySelector("input[name=name]")?.value, "Review");
  assert.equal(environment.container.querySelector("input[name=name]"), globalThis.document.activeElement);
  assert.equal(
    warnings.some((warning) => warning.includes("changing an uncontrolled input to be controlled")),
    false,
    "the real Input component must not receive undefined before StateForm data is applied"
  );
});
