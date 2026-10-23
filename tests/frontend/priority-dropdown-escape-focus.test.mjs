import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { React, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const hooksRoot = path.join(repositoryRoot, "packages/hooks");
const typescript = webRequire("typescript");
const packageStubs = {
  "@plane/constants":
    "export const ISSUE_PRIORITIES = [{ key: 'urgent', title: 'Urgent' }, { key: 'high', title: 'High' }, { key: 'medium', title: 'Medium' }, { key: 'low', title: 'Low' }, { key: 'none', title: 'No priority' }];",
  "@plane/i18n": `export const useTranslation = () => ({ t: key => ({ "common.priority": "Priority", search: "Search" })[key] ?? key });`,
  "@plane/propel/icons":
    "export const CheckIcon = () => null; export const PriorityIcon = () => null; export const ChevronDownIcon = () => null; export const SearchIcon = () => null;",
  "@plane/utils": "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');",
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    const sourceModules = {
      "@plane/hooks": "packages/hooks/src/use-outside-click-detector.tsx",
      "@plane/propel/tooltip": "packages/propel/src/tooltip/index.ts",
      "@plane/ui": "packages/ui/src/dropdowns/combo-box.tsx",
    };
    if (sourceModules[specifier]) {
      return nextResolve(pathToFileURL(path.join(repositoryRoot, sourceModules[specifier])).href, context);
    }
    if (specifier === "./use-platform-os" && context.parentURL?.endsWith("/core/hooks/use-dropdown.ts")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const usePlatformOS = () => ({ isMobile: true });")}`,
      };
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    if (specifier === "@headlessui/react") {
      const webRoot = path.join(fileURLToPath(new URL("../../", import.meta.url)), "apps/web");
      return nextResolve(specifier, { ...context, parentURL: pathToFileURL(path.join(webRoot, "index.js")).href });
    }
    return nextResolve(specifier, context);
  },
  load(url, context, nextLoad) {
    if (url.startsWith(pathToFileURL(hooksRoot).href) && /\.tsx?$/.test(url)) {
      const source = readFileSync(fileURLToPath(url), "utf8");
      return {
        format: "module",
        shortCircuit: true,
        source: typescript.transpileModule(source, {
          compilerOptions: {
            module: typescript.ModuleKind.ESNext,
            target: typescript.ScriptTarget.ES2022,
            jsx: typescript.JsxEmit.ReactJSX,
          },
        }).outputText,
      };
    }
    return nextLoad(url, context);
  },
});

const { PriorityDropdown } = await import("../../apps/web/core/components/dropdowns/priority.tsx");

void test("Escape closes the priority dropdown and returns focus to its trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(
    React.createElement(PriorityDropdown, {
      buttonVariant: "transparent-with-text",
      onChange() {},
      value: "medium",
    })
  );

  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  await React.act(async () => trigger.click());
  const search = document.querySelector('input[placeholder="Search"]');
  assert.ok(search);
  search.attachEvent = () => {};
  search.detachEvent = () => {};
  await React.act(async () => search.focus());
  assert.equal(document.activeElement, search);

  await React.act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });

  assert.equal(document.querySelector('input[placeholder="Search"]'), null);
  assert.equal(document.activeElement, trigger);
});

void test("priority options stay interactive and selecting one updates the value", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  let selectedPriority;

  await environment.render(
    React.createElement(PriorityDropdown, {
      buttonVariant: "transparent-with-text",
      onChange(value) {
        selectedPriority = value;
      },
      value: "medium",
    })
  );

  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  await React.act(async () => trigger.click());

  const highOption = [...document.querySelectorAll('[role="option"]')].find((option) =>
    option.textContent.includes("High")
  );
  assert.ok(highOption);
  assert.equal(highOption.closest('[aria-hidden="true"]'), null);

  await React.act(async () => {
    highOption.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }));
  });
  assert.equal(selectedPriority, "high");
});
