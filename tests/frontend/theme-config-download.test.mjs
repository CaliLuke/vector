import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

const toasts = [];
globalThis.vectorThemeDownload = { React, toasts };
registerWebTestModules({
  "@plane/i18n": "export const useTranslation = () => ({ t: (key) => key });",
  "@plane/propel/button": `export const Button = ({ children, type, onClick }) =>
    globalThis.vectorThemeDownload.React.createElement("button", { type, onClick }, children);`,
  "@plane/propel/toast": `export const TOAST_TYPE = { SUCCESS: "success", ERROR: "error" };
    export const setToast = (toast) => globalThis.vectorThemeDownload.toasts.push(toast);`,
});
const { CustomThemeDownloadConfigButton } =
  await import("../../apps/web/core/components/core/theme/download-config-button.tsx");

await test("theme download cleans its resources and reports only failed attempts", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const diagnostics = [];
  window.fetch = async (path, options) => {
    diagnostics.push({ path, body: JSON.parse(options.body), referrerPolicy: options.referrerPolicy });
    throw new Error("diagnostic connection refused");
  };

  const url = "blob:fixture";
  const createURL = t.mock.method(URL, "createObjectURL", () => url);
  const revokeURL = t.mock.method(URL, "revokeObjectURL", () => {});
  let fail = false;
  const download = t.mock.method(window.HTMLAnchorElement.prototype, "click", function () {
    assert.match(this.download, /^vector-theme-\d+\.json$/);
    assert.equal(this.href, url);
    if (fail) throw new Error("private browser details");
  });
  const value = { theme: "custom", primary: "#ffffff", background: "#000000", darkPalette: true };
  await environment.render(React.createElement(CustomThemeDownloadConfigButton, { value }));

  const button = environment.container.querySelector("button");
  await act(async () => button.click());
  assert.deepEqual(diagnostics, []);
  assert.equal(toasts.at(-1).type, "success");
  assert.deepEqual(JSON.parse(await createURL.mock.calls[0].arguments[0].text()), {
    version: "1.0",
    themeName: "Custom Theme",
    primary: value.primary,
    background: value.background,
    darkPalette: true,
  });

  fail = true;
  await act(async () => button.click());
  assert.equal(download.mock.callCount(), 2);
  assert.equal(document.querySelector("a[download]"), null);
  assert.deepEqual(
    revokeURL.mock.calls.map((call) => call.arguments),
    [[url], [url]]
  );
  assert.equal(toasts.at(-1).type, "error");
  assert.deepEqual(diagnostics, [
    {
      path: "/api/diagnostics/browser-action/",
      body: { action: "theme_config_download" },
      referrerPolicy: "no-referrer",
    },
  ]);
});
