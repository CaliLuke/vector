import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act, webRequire, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { startTranslations, finishTranslations } from "./helpers/translation-initialization.mjs";

registerWebModuleHooks();
const providerURL = new URL("../../packages/i18n/src/provider/index.tsx", import.meta.url).href;
const initializationURL = new URL("./helpers/translation-initialization.mjs", import.meta.url).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL === providerURL && specifier === "../core") {
      return { shortCircuit: true, url: initializationURL };
    }
    return nextResolve(specifier, context);
  },
});
const { TranslationProvider } = await import(providerURL);
const { renderToString } = webRequire("react-dom/server");
const { hydrateRoot } = webRequire("react-dom/client");

for (const browserReady of [false, true]) {
  test(`translation provider hydrates a warm server with ${browserReady ? "warm" : "cold"} browser resources`, async () => {
    startTranslations();
    await finishTranslations();
    const content = React.createElement(
      React.Fragment,
      null,
      React.createElement(TranslationProvider, null, React.createElement("p", null, "Translated content")),
      React.createElement("link", { rel: "modulepreload", href: "/entry.js" })
    );
    const serverHTML = renderToString(content);
    startTranslations();
    if (browserReady) await finishTranslations();

    const environment = await createReactDomTestEnvironment();
    const hydrationContainer = document.createElement("div");
    hydrationContainer.innerHTML = serverHTML;
    document.body.append(hydrationContainer);
    const originalLink = hydrationContainer.querySelector("link");
    const errors = [];
    const originalError = console.error;
    console.error = (...args) => errors.push(args.join(" "));
    let root;
    try {
      await act(async () => {
        root = hydrateRoot(hydrationContainer, content, {
          onRecoverableError: (error) => errors.push(error.message),
        });
      });
      assert.deepEqual(errors, [], "server and initial client render must agree");
      assert.equal(hydrationContainer.querySelector("link"), originalLink, "hydration must preserve the server DOM");
      if (!browserReady) {
        assert.equal(hydrationContainer.querySelector("p"), null, "wait for translation loading");
        await act(async () => finishTranslations());
      }
      assert.equal(hydrationContainer.querySelector("p")?.textContent, "Translated content");
    } finally {
      if (root) await act(async () => root.unmount());
      console.error = originalError;
      hydrationContainer.remove();
      await environment.unmount();
    }
  });
}
