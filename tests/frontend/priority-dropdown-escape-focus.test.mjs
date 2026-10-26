import { setPositioningGeometry } from "./helpers/positioning.mjs";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, webRequire } from "./helpers/react.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { openSearchCombobox } from "./helpers/combobox.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.propertyDropdownReact = React;

const repositoryRoot = fileURLToPath(new URL("../../../", import.meta.resolve("./helpers/web-module-hooks.mjs")));
const hooksRoot = path.join(repositoryRoot, "packages/hooks");
const typescript = webRequire("typescript");
const packageStubs = {
  "@tanstack/react-router": "export const useParams=()=>({workspaceSlug:'acme'});",
  "@plane/constants":
    "export const ISSUE_PRIORITIES = [{ key: 'urgent', title: 'Urgent' }, { key: 'high', title: 'High' }, { key: 'medium', title: 'Medium' }, { key: 'low', title: 'Low' }, { key: 'none', title: 'No priority' }];",
  "@plane/i18n": `export const useTranslation = () => ({ t: key => ({ "common.priority": "Priority", search: "Search" })[key] ?? key });`,
  "@plane/propel/icons":
    "export const CheckIcon = () => null; export const PriorityIcon = () => null; export const ChevronDownIcon = () => null; export const SearchIcon = () => null; export const EstimatePropertyIcon=()=>null; export const IntakeStateGroupIcon=()=>null; export const StateGroupIcon=()=>null; export const ProjectIcon=()=>null;",
  "@plane/propel/emoji-icon-picker": "export const Logo=()=>null;",
  "@plane/utils/datetime": "export const convertMinutesToHoursMinutesString=value=>String(value);",
  "@plane/utils/common":
    "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    const sourceModules = {
      "@plane/hooks": "packages/hooks/src/use-outside-click-detector.tsx",
      "@plane/propel/tooltip": "packages/propel/src/tooltip/index.ts",
      "@plane/ui": "packages/ui/src/dropdowns/combo-box.tsx",
    };
    if (specifier === "@/hooks/use-estimates")
      return {
        shortCircuit: true,
        url:
          "data:text/javascript," +
          encodeURIComponent(
            `export const useProjectEstimates=()=>({activeEstimate:{id:'effort',type:'points'},points:[{id:'point-3',value:'3',key:1}],isPending:false,isError:false,isAccessDenied:false,retry:async()=>{}});`
          ),
      };
    if (specifier === "@plane/ui")
      return {
        shortCircuit: true,
        url:
          "data:text/javascript," +
          encodeURIComponent(
            `export {ComboDropDown,ComboSearchInput} from '${pathToFileURL(path.join(repositoryRoot, sourceModules[specifier])).href}';export const Spinner=()=>null;`
          ),
      };
    if (sourceModules[specifier]) {
      return nextResolve(pathToFileURL(path.join(repositoryRoot, sourceModules[specifier])).href, context);
    }
    if (specifier === "@headlessui/react") {
      const webRoot = path.join(repositoryRoot, "apps/web");
      return nextResolve(specifier, { ...context, parentURL: pathToFileURL(path.join(webRoot, "index.js")).href });
    }
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
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

let LoadedPriorityDropdown;
async function loadPriorityDropdown() {
  LoadedPriorityDropdown ??= (await import("../../apps/web/core/components/dropdowns/priority.tsx")).PriorityDropdown;
  return LoadedPriorityDropdown;
}

async function assertPropertySearchLifecycle(t, load, props, searchText, indexedElements = 0) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const Component = await load();
  await environment.render(
    React.createElement(Component, { buttonVariant: "transparent-with-text", tabIndex: 7, ...props })
  );

  assert.equal(
    environment.container.querySelectorAll('[tabindex="7"]').length,
    indexedElements,
    "preserve each control's original tab stops"
  );
  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  setPositioningGeometry(trigger);
  const search = await openSearchCombobox(trigger);
  const menu = document.querySelector('[role="listbox"]');
  assert.equal(menu.getAttribute("data-popper-placement"), "bottom-start");
  assert.equal(
    menu.style.transform,
    "translate(400px, 132px)",
    "the menu opens below its trigger, away from the page origin"
  );

  await React.act(async () => {
    const setInputValue = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set?.bind(search);
    assert.ok(setInputValue);
    setInputValue(searchText);
    search.dispatchEvent(new Event("input", { bubbles: true }));
  });
  assert.equal(search.value, searchText);
  let escapedAtDocument = false;
  const observeEscape = (event) => {
    if (event.key === "Escape") escapedAtDocument = true;
  };
  document.addEventListener("keydown", observeEscape);
  let clearSearchEvent;
  await React.act(async () => {
    clearSearchEvent = new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true });
    search.dispatchEvent(clearSearchEvent);
  });
  document.removeEventListener("keydown", observeEscape);
  assert.equal(clearSearchEvent.defaultPrevented, true);
  assert.equal(escapedAtDocument, false);
  assert.equal(search.value, "");
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.equal(document.activeElement, search);

  await React.act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });

  assert.equal(document.querySelector('input[role="combobox"]'), null);
  assert.equal(document.activeElement, trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  const reopened = await openSearchCombobox(trigger);
  assert.equal(reopened.value, "", "reopening starts a fresh search and focuses its input");
}
void test("Escape clears priority search before native close and restores focus", async (t) =>
  assertPropertySearchLifecycle(t, loadPriorityDropdown, { onChange() {}, value: "medium" }, "urgent"));
void test("estimate dropdown shares focus, Escape, clear and reopen search lifecycle", async (t) => {
  await assertPropertySearchLifecycle(
    t,
    async () => (await import("../../apps/web/core/components/dropdowns/estimate.tsx")).EstimateDropdown,
    { onChange() {}, value: "point-3", projectId: "project" },
    "3",
    1
  );
});
void test("intake dropdown shares focus, Escape and search reset", async (t) => {
  let closed = 0;
  const state = { id: "triage", name: "Triage", group: "triage", default: true };
  await assertPropertySearchLifecycle(
    t,
    async () =>
      (await import("../../apps/web/core/components/dropdowns/intake-state/base.tsx")).WorkItemStateDropdownBase,
    {
      onChange() {},
      value: "triage",
      projectId: "project",
      stateIds: [state.id],
      getStateById: () => state,
      onClose: () => closed++,
    },
    "triage"
  );
  assert.equal(closed, 1);
});

void test("project dropdown opens a positioned menu and restores search focus", async (t) => {
  await assertPropertySearchLifecycle(
    t,
    async () => (await import("../../apps/web/core/components/dropdowns/project/base.tsx")).ProjectDropdownBase,
    {
      onChange() {},
      value: "project",
      multiple: false,
      projectIds: ["project"],
      getProjectById: () => ({ id: "project", name: "Project" }),
    },
    "Project",
    1
  );
});

void test("state picker stays controlled while its default state loads", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const StateDropdown = (await import("../../apps/web/core/components/dropdowns/state/base.tsx"))
    .WorkItemStateDropdownBase;
  const defaultState = { id: "started", name: "Started", group: "started", color: "blue", default: true };
  const consoleErrors = [];
  const originalConsoleError = console.error;
  console.error = (...args) => consoleErrors.push(args.map(String).join(" "));

  try {
    await environment.render(
      React.createElement(StateDropdown, {
        buttonVariant: "border-with-text",
        value: "",
        onChange() {},
        stateIds: [],
        getStateById: () => defaultState,
      })
    );
    await environment.render(
      React.createElement(StateDropdown, {
        buttonVariant: "border-with-text",
        value: "",
        onChange() {},
        stateIds: [defaultState.id],
        getStateById: (id) => (id === defaultState.id ? defaultState : undefined),
      })
    );
  } finally {
    console.error = originalConsoleError;
  }

  assert.equal(
    consoleErrors.some((message) => message.includes("uncontrolled to controlled")),
    false,
    "the Combobox receives a stable controlled empty value until a default state is available"
  );
  assert.match(environment.container.textContent, /Started/);
});

void test("priority options stay interactive and selecting one updates the value", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const PriorityDropdown = await loadPriorityDropdown();
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
  await clickWithPointer(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");

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
