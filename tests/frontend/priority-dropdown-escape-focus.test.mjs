import { setPositioningGeometry } from "./helpers/positioning.mjs";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { registerHooks } from "./helpers/module-hooks.mjs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { openSearchCombobox } from "./helpers/combobox.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { transformFrontendSource } from "./helpers/source-transform.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.propertyDropdownReact = React;
globalThis.propertyDropdownEstimates = {
  activeEstimate: { id: "effort", type: "points" },
  points: [{ id: "point-3", value: "3", key: 1 }],
  isPending: false,
  isError: false,
  isAccessDenied: false,
  retry: async () => {},
};

const repositoryRoot = fileURLToPath(new URL("../../../", import.meta.resolve("./helpers/web-module-hooks.mjs")));
const hooksRoot = path.join(repositoryRoot, "packages/hooks");
const packageStubs = {
  "@tanstack/react-router": "export const useParams=()=>({workspaceSlug:'acme'});",
  "@plane/constants":
    "export const ISSUE_PRIORITIES = [{ key: 'urgent', title: 'Urgent' }, { key: 'high', title: 'High' }, { key: 'medium', title: 'Medium' }, { key: 'low', title: 'Low' }, { key: 'none', title: 'No priority' }];",
  "@plane/i18n": `export const useTranslation = () => ({ t: key => ({ "common.priority": "Priority", search: "Search" })[key] ?? key });`,
  "@plane/propel/icons":
    "export const CheckIcon = () => null; export const PriorityIcon = () => null; export const ChevronDownIcon = () => null; export const SearchIcon = () => null; export const EstimatePropertyIcon=()=>null; export const IntakeStateGroupIcon=()=>null; export const StateGroupIcon=()=>null; export const ProjectIcon=()=>null;",
  "@plane/propel/emoji-icon-picker": "export const Logo=()=>null;",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    const sourceModules = {
      "@plane/propel/tooltip": "packages/propel/src/tooltip/index.ts",
      "@plane/ui": "packages/ui/src/dropdowns/combo-box.tsx",
    };
    if (specifier === "@plane/hooks")
      return {
        shortCircuit: true,
        url:
          "data:text/javascript," +
          encodeURIComponent(
            ["use-outside-click-detector", "use-platform-os"]
              .map(
                (name) =>
                  `export * from ${JSON.stringify(pathToFileURL(path.join(hooksRoot, "src", `${name}.tsx`)).href)};`
              )
              .join("\n")
          ),
      };
    if (specifier === "@/hooks/use-estimates")
      return {
        shortCircuit: true,
        url:
          "data:text/javascript," +
          encodeURIComponent(`export const useProjectEstimates=()=>globalThis.propertyDropdownEstimates;`),
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
        source: transformFrontendSource(source, {
          format: "esm",
          target: "es2022",
          tsconfigRaw: { compilerOptions: { useDefineForClassFields: true } },
          loader: url.endsWith(".tsx") ? "tsx" : "ts",
          jsx: "automatic",
          sourcefile: fileURLToPath(url),
        }),
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

void test("time estimates format minutes and clearing the native selection emits an omitted estimate", async (t) => {
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.map(String).join(" ")));
  const previous = globalThis.propertyDropdownEstimates;
  globalThis.propertyDropdownEstimates = {
    ...previous,
    activeEstimate: { id: "time", type: "time" },
    points: [
      { id: "time-90", value: "90", key: 1 },
      { id: "", value: "invalid summary", key: 2 },
    ],
  };
  t.after(() => {
    globalThis.propertyDropdownEstimates = previous;
  });
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { EstimateDropdown } = await import("../../apps/web/core/components/dropdowns/estimate.tsx");
  const changes = [];
  function Subject() {
    const [value, setValue] = React.useState("time-90");
    return React.createElement(EstimateDropdown, {
      projectId: "project",
      buttonVariant: "transparent-with-text",
      placeholder: "Choose estimate",
      value,
      onChange: (next) => {
        changes.push(next);
        setValue(next);
      },
    });
  }
  await environment.render(React.createElement(Subject));
  const trigger = environment.container.querySelector("button");
  assert.equal(trigger.textContent.trim(), "1h 30m");
  await openSearchCombobox(trigger);
  const options = [...document.querySelectorAll('[role="option"]')];
  assert.deepEqual(
    options.map((option) => option.textContent.trim()),
    ["project_settings.estimates.no_estimate", "1h 30m"]
  );
  await React.act(async () => options[0].dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 })));
  assert.deepEqual(changes, [undefined]);
  assert.equal(trigger.textContent.trim(), "Choose estimate");
  assert.deepEqual(errors, []);
});

void test("multi-project selection excludes the source and hidden projects and can be emptied through native options", async (t) => {
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.map(String).join(" ")));
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ProjectDropdownBase } = await import("../../apps/web/core/components/dropdowns/project/base.tsx");
  const projects = {
    current: { id: "current", name: "Current" },
    first: { id: "first", name: "First", logo_props: { type: "emoji", value: "128640" } },
    second: { id: "second", name: "Second" },
    hidden: { id: "hidden", name: "Hidden" },
  };
  const changes = [];
  function Subject() {
    const [value, setValue] = React.useState(["first"]);
    return React.createElement(ProjectDropdownBase, {
      multiple: true,
      buttonVariant: "transparent-with-text",
      value,
      placeholder: "Pick projects",
      currentProjectId: "current",
      projectIds: Object.keys(projects),
      getProjectById: (id) => projects[id],
      renderCondition: (id) => id !== "hidden",
      onChange: (next) => {
        changes.push(next);
        setValue(next);
      },
    });
  }
  await environment.render(React.createElement(Subject));
  const trigger = environment.container.querySelector("button");
  assert.equal(trigger.textContent, "First");
  await openSearchCombobox(trigger);
  const options = () => [...environment.container.ownerDocument.querySelectorAll('[role="option"]')];
  assert.deepEqual(
    options().map((option) => option.textContent),
    ["First", "Second"]
  );
  async function choose(label) {
    if (trigger.getAttribute("aria-expanded") !== "true") await openSearchCombobox(trigger);
    const option = options().find((item) => item.textContent === label);
    assert.ok(option);
    await React.act(async () => option.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 })));
  }
  await choose("Second");
  assert.deepEqual(changes.at(-1), ["first", "second"]);
  assert.equal(trigger.textContent, "2 projects");
  await choose("First");
  assert.equal(trigger.textContent, "Second");
  await choose("Second");
  assert.deepEqual(changes.at(-1), []);
  assert.equal(trigger.textContent, "Pick projects");
  assert.deepEqual(errors, []);
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

for (const variant of ["border", "background", "transparent"]) {
  void test(`priority ${variant} controls retain text, compact and urgent states`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const PriorityDropdown = await loadPriorityDropdown();
    const render = (suffix, value, extra = {}) =>
      environment.render(
        React.createElement(PriorityDropdown, {
          buttonVariant: `${variant}-${suffix}`,
          value,
          onChange() {},
          ...extra,
        })
      );
    await render("with-text", "urgent");
    const trigger = environment.container.querySelector("button");
    assert.equal(trigger.textContent, "Urgent");
    assert.ok(trigger.querySelector(".border-priority-urgent"));
    await render("without-text", "urgent");
    assert.equal(trigger.textContent, "");
    assert.ok(trigger.querySelector(".border-priority-urgent"));
    await render("with-text", null, { placeholder: "Choose priority" });
    assert.equal(trigger.textContent, variant === "border" ? "Choose priority" : "Priority");
    await render("with-text", "low", { disabled: true });
    assert.equal(trigger.disabled, true);
    assert.equal(trigger.textContent, "Low");
  });
}
