import { createRequire, registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import { act } from "./react.mjs";
import { observeNativeForms } from "./native-form-observer.mjs";
import { registerAdminModuleHooks } from "./web-module-hooks.mjs";
import { resolvePackageStub } from "./module-stubs.mjs";

export const adminRequire = createRequire(new URL("../../../apps/admin/package.json", import.meta.url));
export const formToasts = [];
globalThis.adminFormToasts = formToasts;
const stubs = {
  "@plane/propel/button": `import React from "react"; export const getButtonStyling = () => "";
    export const Button = ({children, loading, disabled, variant, size, ...props}) => React.createElement("button", {...props, disabled: disabled || loading}, children);`,
  "@plane/propel/toast": `export const TOAST_TYPE = {SUCCESS: "success", ERROR: "error"}; export const setToast = value => globalThis.adminFormToasts.push(value);`,
  "@plane/ui": `import React from "react";
    export const Spinner = () => React.createElement("span");
    export const PasswordStrengthIndicator = () => null;
    export { AuthInput, AuthEmailInput } from ${JSON.stringify(new URL("../../../packages/ui/src/auth-form/auth-input.tsx", import.meta.url).href)};
    export const Checkbox = ({iconClassName, ...props}) => React.createElement("input", {...props, type: "checkbox"});
    export const Loader = ({children}) => React.createElement("div", null, children); Loader.Item = () => React.createElement("div");
    export const Input = React.forwardRef(({hasError, inputSize, ...props}, ref) => React.createElement("input", {...props, ref}));
    export const ToggleSwitch = ({label, value, onChange, disabled}) => React.createElement("button", {type: "button", role: "switch", "aria-label": label, "aria-checked": value, disabled, onClick: () => onChange(!value)});`,
  "@/components/common/copy-field": `import React from "react"; export const CopyField = ({url, description}) => React.createElement("div", null, url, description);`,
  "@/components/common/confirm-discard-modal": `import React from "react"; export const ConfirmDiscardModal = ({isOpen}) => isOpen ? React.createElement("div", {role: "dialog"}, "Unsaved changes") : null;`,
};

export function registerAdminFormHooks() {
  registerAdminModuleHooks();
  const packageURLs = new Map(["react"].map((name) => [name, pathToFileURL(adminRequire.resolve(name)).href]));
  registerHooks({
    resolve(specifier, context, nextResolve) {
      if (packageURLs.has(specifier)) return { shortCircuit: true, url: packageURLs.get(specifier) };
      if (specifier.endsWith("?url"))
        return { shortCircuit: true, url: 'data:text/javascript,export default "asset.svg"' };
      return resolvePackageStub(specifier, stubs, context, nextResolve);
    },
  });
}

export function setupAdminSettingsFormTest(sourceFiles, specifier = "@/components/common/settings-form") {
  registerAdminFormHooks();
  return observeNativeForms(sourceFiles, { [specifier]: ["useSettingsForm"] });
}

export async function changeInput(input, value) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
export async function submitAndWait(button, formApi) {
  let sawSubmitting = false;
  let subscription;
  const submission = new Promise((resolve) => {
    subscription = formApi.store.subscribe(() => {
      if (formApi.state.isSubmitting) sawSubmitting = true;
      else if (sawSubmitting) resolve();
    });
  });
  let timeout;
  try {
    await act(async () => {
      button.click();
      await Promise.race([
        submission,
        new Promise((_, reject) => {
          timeout = setTimeout(() => reject(new Error("form submission did not settle")), 3000);
        }),
      ]);
    });
  } finally {
    clearTimeout(timeout);
    subscription.unsubscribe();
  }
  if (formApi.state.isSubmitting) throw new Error("form submission did not finish");
}
export function formButton(container, label) {
  return [...container.querySelectorAll("button")].find((button) => button.textContent === label);
}
export const settleForm = () => act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
