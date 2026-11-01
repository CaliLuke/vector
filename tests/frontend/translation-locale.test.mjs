import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { pathToFileURL } from "node:url";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const i18nRequire = createRequire(new URL("../../packages/i18n/package.json", import.meta.url));
const { createInstance } = i18nRequire("i18next");
const { I18nextProvider } = await import(
  new URL("./dist/es/index.js", pathToFileURL(i18nRequire.resolve("react-i18next/package.json"))).href
);
const { useTranslation } = await import("../../packages/i18n/src/hooks/use-translation.ts");
function LocaleLabel() {
  const { currentLocale, t } = useTranslation();
  return React.createElement("p", null, `${currentLocale}: ${t("greeting")}`);
}
for (const [language, expected] of [
  ["fr", "fr: Bonjour"],
  ["fr-CA", "fr: Bonjour"],
  ["unknown", "en: Hello"],
]) {
  void test(`translation locale ${language} resolves to a supported picker value`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const i18n = createInstance();
    await i18n.init({
      lng: language,
      fallbackLng: "en",
      resources: {
        en: { translation: { greeting: "Hello" } },
        fr: { translation: { greeting: "Bonjour" } },
      },
    });
    await env.render(React.createElement(I18nextProvider, { i18n }, React.createElement(LocaleLabel)));
    assert.equal(env.container.textContent, expected);
  });
}
