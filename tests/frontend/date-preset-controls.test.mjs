import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

const preferencesContext = React.createContext(null);
globalThis.datePresetTest = { React, preferencesContext };
test.after(() => {
  delete globalThis.datePresetTest;
});
registerWebTestModules({
  "@/providers/intake-preferences":
    "const {React,preferencesContext} = globalThis.datePresetTest; export const useIntakePreferences = () => React.useContext(preferencesContext);",
  "@/components/core/filters/date-filter-modal": `const {React} = globalThis.datePresetTest;
    export function DateFilterModal({title, onSelect, handleClose}) {
      return React.createElement('button', {onClick: () => {onSelect(['2026-10-01;after']); handleClose();}}, 'Choose '+title);
    }`,
});
const { DatePresetFilter } = await import("../../apps/web/core/components/core/filters/date-preset-filter.tsx");
const { FilterDate } = await import("../../apps/web/core/components/inbox/inbox-filter/filters/date.tsx");
const buttonWithText = (container, text) =>
  [...container.querySelectorAll("button")].find((button) => button.textContent === text);

await test("custom dates toggle off without dropping relative presets; search and date selection remain available", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const changes = [];
  const props = {
    title: "Created date",
    searchQuery: "",
    options: [{ name: "Last week", value: "1_weeks;before;fromnow" }],
    appliedFilters: ["2026-10-01;after", "2026-10-31;before", "1_weeks;before;fromnow"],
    handleUpdate: (value) => changes.push(value),
  };
  await environment.render(React.createElement(DatePresetFilter, props));
  await act(async () => buttonWithText(environment.container, "Custom").click());
  assert.deepEqual(changes, [["1_weeks;before;fromnow"]]);
  await environment.render(React.createElement(DatePresetFilter, { ...props, appliedFilters: [] }));
  await act(async () => buttonWithText(environment.container, "Custom").click());
  await act(async () => buttonWithText(environment.container, "Choose Created date").click());
  assert.deepEqual(changes.at(-1), ["2026-10-01;after"]);
  assert.equal(buttonWithText(environment.container, "Choose Created date"), undefined);
  await act(async () => buttonWithText(environment.container, "Last week").click());
  assert.equal(changes.at(-1), "1_weeks;before;fromnow");
  await environment.render(React.createElement(DatePresetFilter, { ...props, searchQuery: "missing" }));
  assert.match(environment.container.textContent, /No matches found/);
  await act(async () => environment.container.querySelector("button").click());
  assert.doesNotMatch(environment.container.textContent, /No matches found/);
});

await test("intake date presets replace single choices and retain the selected field for custom dates", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const changes = [];
  function Intake() {
    const [filters, setFilters] = React.useState({ updated_at: [] });
    const setFilter = (key, value) => {
      changes.push([key, value]);
      setFilters({ [key]: value });
    };
    return React.createElement(
      preferencesContext.Provider,
      { value: { filters, setFilter } },
      React.createElement(FilterDate, { filterKey: "updated_at", label: "Last updated date", searchQuery: "" })
    );
  }
  await environment.render(React.createElement(Intake));
  const preset = [...environment.container.querySelectorAll("button")][1];
  await act(async () => preset.click());
  assert.equal(changes.at(-1)[0], "updated_at");
  assert.equal(changes.at(-1)[1].length, 1);
  await act(async () => preset.click());
  assert.deepEqual(changes.at(-1), ["updated_at", []]);
  await act(async () => buttonWithText(environment.container, "Custom").click());
  await act(async () => buttonWithText(environment.container, "Choose Last updated date").click());
  assert.deepEqual(changes.at(-1), ["updated_at", ["2026-10-01;after"]]);
  await act(async () => buttonWithText(environment.container, "Custom").click());
  assert.deepEqual(changes.at(-1), ["updated_at", []]);
});
