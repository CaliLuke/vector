import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

globalThis.React = React;
globalThis.sharedDropdownFilterTest = {
  members: {
    alice: { id: "alice", display_name: "Alice", avatar_url: "alice.png" },
    bob: { id: "bob", display_name: "Bob", avatar_url: "bob.png" },
    me: { id: "me", display_name: "Zoe", avatar_url: "me.png" },
  },
  currentUserId: "me",
  inboxFilters: { assignees: ["bob"] },
  inboxUpdates: [],
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
  "@plane/utils/file": `export const getFileURL = value => value;`,
  "@plane/utils/datetime": `export const isInDateFormat = value => /^\\d{4}-\\d{2}-\\d{2}$/.test(value);`,
  "@/providers/intake-preferences": `export const useIntakePreferences = () => ({
    filters: globalThis.sharedDropdownFilterTest.inboxFilters,
    setFilter: (key, value) => {
      globalThis.sharedDropdownFilterTest.inboxUpdates.push([key, value]);
      if (value === undefined) delete globalThis.sharedDropdownFilterTest.inboxFilters[key];
      else globalThis.sharedDropdownFilterTest.inboxFilters[key] = value;
    },
  });`,
  "@/hooks/use-current-user": `export const useCurrentUser = () => ({ data: { id: globalThis.sharedDropdownFilterTest.currentUserId } });`,
  "@/hooks/use-member-rosters": `export const useWorkspaceMemberRoster = () => ({
    membersById: new Map(Object.entries(globalThis.sharedDropdownFilterTest.members).map(([id, member]) => [id, { member }])),
  });`,
  "@/components/core/filters/date-filter-modal": `export function DateFilterModal({ title, isOpen, onSelect }) {
    if (!isOpen) return null;
    return globalThis.React.createElement("div", null,
      globalThis.React.createElement("output", null, title),
      globalThis.React.createElement("button", { type: "button", onClick: () => onSelect(["2024-01-05;after"]) }, "Apply custom date")
    );
  }`,
};

registerWebTestModules(packageStubs);

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

  const { FilterMember } = await import("../../apps/web/core/components/inbox/inbox-filter/filters/members.tsx");
  globalThis.sharedDropdownFilterTest.inboxUpdates.length = 0;
  await environment.render(
    React.createElement(FilterMember, {
      filterKey: "assignees",
      label: "Assignees",
      memberIds: ["alice", "bob", "me", "unknown"],
      searchQuery: "ali",
      workspaceSlug: "acme",
    })
  );
  assert.match(environment.container.textContent, /Assignees \(1\)/);
  assert.match(environment.container.textContent, /Alice/);
  assert.doesNotMatch(environment.container.textContent, /Bob|You|unknown/);
  const filteredAvatar = environment.container.querySelector('[data-avatar="Alice"]');
  assert.ok(filteredAvatar, "inbox options retain the shared member avatar");
  await React.act(async () => filteredAvatar.closest("button").click());
  assert.deepEqual(globalThis.sharedDropdownFilterTest.inboxUpdates, [["assignees", ["bob", "alice"]]]);

  await environment.render(
    React.createElement(FilterMember, {
      filterKey: "assignees",
      label: "Assignees",
      memberIds: ["alice", "bob", "me", "unknown"],
      searchQuery: "",
      workspaceSlug: "acme",
    })
  );
  assert.match(environment.container.textContent, /You/);
  assert.doesNotMatch(environment.container.textContent, /unknown/);
  assert.equal(environment.container.querySelectorAll("[data-avatar]").length, 3);
  const selectedBob = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Bob")
  );
  await React.act(async () => selectedBob.click());
  assert.deepEqual(globalThis.sharedDropdownFilterTest.inboxUpdates[1], ["assignees", ["alice"]]);
});

await test("date preset filter retains preset selection and opens the titled custom date flow", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { DatePresetFilter } = await import("../../apps/web/core/components/core/filters/date-preset-filter.tsx");
  const updates = [];
  const props = {
    appliedFilters: null,
    handleUpdate: (value) => updates.push(value),
    searchQuery: "",
  };

  await environment.render(React.createElement(DatePresetFilter, { ...props, title: "Due date" }));
  assert.match(environment.container.textContent, /Due date/);
  const preset = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("1 week from now")
  );
  assert.ok(preset, "date preset should render as an interactive button");
  await React.act(async () => preset.click());
  assert.deepEqual(updates, ["1_weeks;after;fromnow"]);

  await environment.render(
    React.createElement(DatePresetFilter, { ...props, title: "Due date", appliedFilters: ["1_weeks;after;fromnow"] })
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

  await environment.render(React.createElement(DatePresetFilter, { ...props, title: "Start date" }));
  assert.match(environment.container.textContent, /Start date/);
});

await test("single-choice date filters use their own searchable presets", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { DatePresetFilter } = await import("../../apps/web/core/components/core/filters/date-preset-filter.tsx");
  const updates = [];
  const props = {
    title: "Created date",
    appliedFilters: ["today;custom;custom"],
    options: [{ name: "Today", value: "today;custom;custom" }],
    multiple: false,
    handleUpdate: (value) => updates.push(value),
    searchQuery: "",
  };
  await environment.render(React.createElement(DatePresetFilter, props));
  const today = [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "Today");
  assert.ok(today);
  assert.match(today.firstElementChild.className, /rounded-full/);
  assert.match(today.firstElementChild.className, /bg-accent-primary/);
  assert.doesNotMatch(environment.container.textContent, /week from now/);
  await React.act(async () => today.click());
  assert.deepEqual(updates, ["today;custom;custom"]);

  await environment.render(React.createElement(DatePresetFilter, { ...props, searchQuery: "yesterday" }));
  assert.match(environment.container.textContent, /No matches found/);
  assert.deepEqual(updates, ["today;custom;custom"]);
});
