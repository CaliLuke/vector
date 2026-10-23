import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.React = React;
globalThis.sharedDropdownFilterTest = {
  members: {
    alice: { id: "alice", display_name: "Alice", avatar_url: "alice.png" },
    bob: { id: "bob", display_name: "Bob", avatar_url: "bob.png" },
    me: { id: "me", display_name: "Zoe", avatar_url: "me.png" },
  },
  currentUserId: "me",
};

const packageStubs = {
  "@plane/constants": `export const DATE_AFTER_FILTER_OPTIONS = [
    { name: "1 week from now", value: "1_weeks;after;fromnow" },
    { name: "1 month from now", value: "1_months;after;fromnow" },
  ];`,
  "@plane/propel/icons": `export const CheckIcon = () => null;
    export const ChevronDownIcon = () => null;
    export const ChevronUpIcon = () => null;`,
  "@plane/ui": `export const Avatar = ({ name }) => globalThis.React.createElement("span", { "data-avatar": name });
    const LoaderComponent = ({ children }) => globalThis.React.createElement("div", null, children);
    LoaderComponent.Item = () => globalThis.React.createElement("div");
    export const Loader = LoaderComponent;`,
  "@plane/utils": `export const getFileURL = value => value;
    export const isInDateFormat = value => /^\\d{4}-\\d{2}-\\d{2}$/.test(value);`,
  "@/hooks/store/use-member": `export const useMember = () => ({ getUserDetails: id => globalThis.sharedDropdownFilterTest.members[id] });`,
  "@/hooks/store/user": `export const useUser = () => ({ data: { id: globalThis.sharedDropdownFilterTest.currentUserId } });`,
  "@/components/core/filters/date-filter-modal": `export function DateFilterModal({ title, isOpen, onSelect }) {
    if (!isOpen) return null;
    return globalThis.React.createElement("div", null,
      globalThis.React.createElement("output", null, title),
      globalThis.React.createElement("button", { type: "button", onClick: () => onSelect(["2024-01-05;after"]) }, "Apply custom date")
    );
  }`,
  "mobx-react": `export const observer = component => component;`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
  },
});

await test("member filter wrappers retain titles, selected ordering, user label, and filtered selection", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { FilterLead } = await import("../../apps/web/core/components/modules/dropdowns/filters/lead.tsx");
  const { FilterMembers } = await import("../../apps/web/core/components/project/dropdowns/filters/members.tsx");
  const updates = [];

  const props = {
    appliedFilters: ["bob"],
    handleUpdate: (value) => updates.push(value),
    memberIds: ["alice", "bob", "me"],
    searchQuery: "",
  };
  await environment.render(React.createElement(FilterLead, props));
  assert.match(environment.container.textContent, /Lead \(1\)/);
  const memberButtons = [...environment.container.querySelectorAll("button")].filter((button) =>
    ["Bob", "You", "Alice"].some((name) => button.textContent.includes(name))
  );
  assert.deepEqual(
    memberButtons.map((button) => button.textContent.trim()),
    ["Bob", "You", "Alice"],
    "selected members remain first, followed by the current user and alphabetical options"
  );
  assert.equal(memberButtons[0].tagName, "BUTTON", "member choices remain native keyboard-focusable buttons");
  await React.act(async () => {
    memberButtons[0].focus();
    memberButtons[0].dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
    // jsdom does not synthesize native keyboard activation.
    memberButtons[0].click();
  });
  assert.deepEqual(updates, ["bob"]);
  assert.match(memberButtons[0].firstElementChild.className, /bg-accent-primary/);

  await environment.render(React.createElement(FilterLead, { ...props, appliedFilters: ["alice"] }));
  const updatedButtons = [...environment.container.querySelectorAll("button")].filter((button) =>
    ["Bob", "You", "Alice"].some((name) => button.textContent.includes(name))
  );
  assert.deepEqual(
    updatedButtons.map((button) => button.textContent.trim()),
    ["Alice", "You", "Bob"],
    "selection changes reorder options without needing a search change"
  );
  await environment.render(React.createElement(FilterLead, { ...props, memberIds: ["alice"] }));
  assert.doesNotMatch(environment.container.textContent, /Bob|You/, "removed workspace members disappear immediately");

  await environment.render(
    React.createElement(FilterLead, { ...props, appliedFilters: ["alice"], searchQuery: "ali" })
  );
  assert.match(environment.container.textContent, /Lead \(1\)/);
  assert.match(environment.container.textContent, /Alice/);
  assert.doesNotMatch(environment.container.textContent, /Bob|You/);

  await environment.render(React.createElement(FilterMembers, props));
  assert.match(environment.container.textContent, /Members \(1\)/);
});

await test("date preset filter retains preset selection and opens the titled custom date flow", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { FilterTargetDate } = await import("../../apps/web/core/components/modules/dropdowns/filters/target-date.tsx");
  const { FilterStartDate } = await import("../../apps/web/core/components/cycles/dropdowns/filters/start-date.tsx");
  const updates = [];
  const props = {
    appliedFilters: null,
    handleUpdate: (value) => updates.push(value),
    searchQuery: "",
  };

  await environment.render(React.createElement(FilterTargetDate, props));
  assert.match(environment.container.textContent, /Due date/);
  const preset = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("1 week from now")
  );
  assert.ok(preset, "date preset should render as an interactive button");
  await React.act(async () => preset.click());
  assert.deepEqual(updates, ["1_weeks;after;fromnow"]);

  await environment.render(
    React.createElement(FilterTargetDate, { ...props, appliedFilters: ["1_weeks;after;fromnow"] })
  );
  const selectedPreset = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("1 week from now")
  );
  assert.match(selectedPreset.firstElementChild.className, /bg-accent-primary/);

  const custom = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Custom")
  );
  await React.act(async () => custom.click());
  assert.equal(environment.container.querySelector("output")?.textContent, "Due date");
  const applyCustomDate = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Apply custom date")
  );
  assert.ok(applyCustomDate, "opening Custom should expose the custom date flow");
  await React.act(async () => applyCustomDate.click());
  assert.deepEqual(updates, ["1_weeks;after;fromnow", ["2024-01-05;after"]]);

  await environment.render(React.createElement(FilterStartDate, props));
  assert.match(environment.container.textContent, /Start date/);
});
