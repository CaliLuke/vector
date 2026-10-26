import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "../utils" && context.parentURL?.endsWith("/breadcrumbs/breadcrumbs.tsx")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const cn = (...classes) => classes.filter(Boolean).join(' ');")}`,
      };
    }
    const moduleSources = {
      "@plane/propel/icons": "export const ChevronRightIcon = () => null;",
      "@plane/propel/tooltip": "export const Tooltip = ({ children }) => children;",
    };
    if (moduleSources[specifier]) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(moduleSources[specifier])}`,
      };
    }
    return nextResolve(specifier, context);
  },
});
const { Breadcrumbs } = await import("../../packages/ui/src/breadcrumbs/breadcrumbs.tsx");

test("loading breadcrumbs keep list keys and restore item navigation", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const errors = [];
  const originalConsoleError = console.error;
  console.error = (...args) => errors.push(args.join(" "));
  t.after(async () => {
    console.error = originalConsoleError;
    await environment.unmount();
  });

  const items = [
    React.createElement(Breadcrumbs.Item, {
      key: "projects",
      component: React.createElement("a", { href: "/projects" }, "Projects"),
    }),
    React.createElement(Breadcrumbs.Item, {
      key: "issues",
      component: React.createElement("a", { href: "/projects/one/issues" }, "Issues"),
    }),
  ];

  await environment.render(React.createElement(Breadcrumbs, { isLoading: true }, items));
  assert.equal(environment.container.querySelectorAll(".animate-pulse").length, 2);

  await environment.render(React.createElement(Breadcrumbs, { isLoading: false }, items));

  assert.deepEqual(
    [...environment.container.querySelectorAll("a")].map((link) => [link.textContent, link.getAttribute("href")]),
    [
      ["Projects", "/projects"],
      ["Issues", "/projects/one/issues"],
    ]
  );
  assert.equal(
    errors.some((message) => message.includes('unique "key" prop')),
    false,
    errors.join("\n")
  );
});

test("mobile breadcrumb back control is keyboard-focusable and activates on click", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(async () => environment.unmount());

  window.innerWidth = 400;
  let backCount = 0;
  const items = [
    React.createElement(Breadcrumbs.Item, { key: "projects", component: "Projects" }),
    React.createElement(Breadcrumbs.Item, { key: "issues", component: "Issues" }),
  ];

  await environment.render(React.createElement(Breadcrumbs, { onBack: () => backCount++ }, items));

  const backButton = environment.container.querySelector('button[aria-label="Go back"]');
  assert.ok(backButton);
  assert.equal(backButton.type, "button");
  backButton.focus();
  assert.equal(document.activeElement, backButton);
  backButton.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
  backButton.click();
  assert.equal(backCount, 1);
});
