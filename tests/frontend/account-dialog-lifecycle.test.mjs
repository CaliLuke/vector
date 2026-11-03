import assert from "node:assert/strict";
import test from "node:test";
import { createRequire } from "./helpers/module-hooks.mjs";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

const { MemoryRouter } = createRequire(new URL("../../apps/admin/package.json", import.meta.url))("react-router");
globalThis.accountDialogReact = React;
registerWebTestModules({
  "@plane/ui": `export {ModalBackdrop} from ${JSON.stringify(new URL("../../packages/ui/src/modals/modal-core.tsx", import.meta.url).href)};`,
  "@plane/propel/button": `const R=globalThis.accountDialogReact;export const getButtonStyling=()=>'';
    export const Button=({variant,size,children,...props})=>R.createElement('button',props,children);`,
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error'};export const setToast=()=>globalThis.accountDialogState.failures++;",
  "@tanstack/react-router":
    "export const useRouter=()=>({navigate:options=>globalThis.accountDialogState.navigations.push(options)});",
  "next-themes": "export const useTheme=()=>({setTheme:value=>globalThis.accountDialogState.themes.push(value)});",
  "@/hooks/use-current-user": `export const useCurrentUser=()=>({data:{email:'person@example.com'}});
    export const useSignOut=()=>({mutateAsync:()=>{globalThis.accountDialogState.signOutCalls++;return globalThis.accountDialogState.request.promise;}});`,
});

const dialog = () => document.querySelector('[role="dialog"]');

async function setup(env, kind) {
  await env.render(null);
  const state = (globalThis.accountDialogState = {
    request: Promise.withResolvers(),
    signOutCalls: 0,
    failures: 0,
    navigations: [],
    themes: [],
    closes: 0,
    recordClose() {
      this.closes++;
    },
  });
  const Component =
    kind === "admin"
      ? (await import("../../apps/admin/components/common/confirm-discard-modal.tsx")).ConfirmDiscardModal
      : (await import("../../apps/web/core/components/onboarding/switch-account-modal.tsx")).SwitchAccountModal;
  function Harness() {
    const [open, setOpen] = React.useState(false);
    const close = () => {
      state.recordClose();
      setOpen(false);
    };
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("button", { onClick: () => setOpen(true) }, "Open account dialog"),
      React.createElement(
        Component,
        kind === "admin"
          ? { isOpen: open, handleClose: close, onDiscardHref: "/settings" }
          : { isOpen: open, onClose: close }
      )
    );
  }
  await env.render(React.createElement(MemoryRouter, null, React.createElement(Harness)));
  const opener = env.container.querySelector("button");
  const open = async () => {
    opener.focus();
    await act(async () => opener.click());
    await waitForReactCondition(() => Boolean(dialog()), "The native account dialog should open");
  };
  return { env, state, opener, dialog, open };
}

async function verifyDismissal(t, env, kind) {
  await t.test(`${kind} account dialog retains native Escape dismissal and restores its opener`, async () => {
    const view = await setup(env, kind);
    assert.equal(Boolean(view.dialog()), false);
    await view.open();
    assert.match(view.dialog().textContent, kind === "admin" ? /unsaved changes/ : /Switch account/);
    if (kind === "admin") assert.equal(view.dialog().querySelector("a").getAttribute("href"), "/settings");
    await act(async () =>
      document.activeElement.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
      )
    );
    await waitForReactCondition(() => !view.dialog(), "Escape should close the native dialog");
    await waitForReactCondition(() => document.activeElement === view.opener, "Focus should return to the opener");
    assert.equal(view.state.closes, 1);
  });
}
await test("account dialogs preserve their native lifecycle in one browser document", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await verifyDismissal(t, env, "admin");
  await verifyDismissal(t, env, "web");

  await t.test(
    "switch account retains sign-out failure for retry and closes only after successful sign-out",
    async () => {
      const view = await setup(env, "web");
      await view.open();
      const submit = () => view.dialog().querySelector("button");
      await act(async () => submit().click());
      assert.equal(submit().disabled, true);
      assert.equal(view.state.signOutCalls, 1);
      await act(async () => view.state.request.reject(Error("offline")));
      assert.equal(view.state.failures, 1);
      assert.equal(submit().disabled, false);
      assert.equal(view.state.closes, 0);
      assert.deepEqual(view.state.navigations, []);
      view.state.request = Promise.withResolvers();
      await act(async () => submit().click());
      await act(async () => view.state.request.resolve());
      await waitForReactCondition(() => !view.dialog(), "Successful sign-out should close the dialog");
      assert.equal(view.state.signOutCalls, 2);
      assert.deepEqual(view.state.themes, ["system"]);
      assert.deepEqual(view.state.navigations, [{ href: "/" }]);
      assert.equal(view.state.closes, 1);
    }
  );
});
