import assert from "node:assert/strict";
import { after, afterEach, test } from "node:test";
import { act, createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const environment = await createReactDomTestEnvironment();

const { BreadcrumbNavigationDropdown } = await import("../../packages/ui/src/breadcrumbs/navigation-dropdown.tsx");
const { BreadcrumbNavigationSearchDropdown } =
  await import("../../packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx");

afterEach(async () => {
  await environment.render(null);
});

after(async () => {
  await environment.unmount();
});

function assertNoNestedButtons(container) {
  for (const button of container.querySelectorAll("button")) {
    assert.equal(button.querySelector("button"), null, "interactive breadcrumb controls must not nest buttons");
  }
}

test("search breadcrumb keeps navigation and menu actions on separate buttons", async () => {
  let navigated = 0;
  let changedTo = "";
  await environment.render(
    React.createElement(BreadcrumbNavigationSearchDropdown, {
      title: "Project alpha",
      selectedItem: "alpha",
      navigationItems: [
        { value: "alpha", query: "Project alpha", content: "Project alpha" },
        { value: "beta", query: "Project beta", content: "Project beta" },
      ],
      handleOnClick: () => navigated++,
      onChange: (value) => (changedTo = value),
    })
  );

  assertNoNestedButtons(environment.container);
  assert.equal(environment.container.querySelectorAll("button").length, 2);
  await act(async () => environment.container.querySelector("button")?.click());
  assert.equal(navigated, 1, "clicking the non-current label should navigate");

  await act(async () => environment.container.querySelector("button[aria-label='Open breadcrumb menu']")?.click());
  const option = [...document.querySelectorAll('[role="option"]')].find((item) =>
    item.textContent.includes("Project beta")
  );
  assert.ok(option, "the chevron trigger should open the searchable project menu");
  await act(async () => option.click());
  assert.equal(changedTo, "beta");
});

test("last search breadcrumb opens its menu when the title is clicked", async () => {
  await environment.render(
    React.createElement(BreadcrumbNavigationSearchDropdown, {
      title: "Project alpha",
      selectedItem: "alpha",
      navigationItems: [{ value: "alpha", query: "Project alpha", content: "Project alpha" }],
      isLast: true,
    })
  );

  assertNoNestedButtons(environment.container);
  assert.equal(environment.container.querySelectorAll("button").length, 1);
  await act(async () => environment.container.querySelector("button")?.click());
  assert.ok(document.querySelector("input[placeholder='Search']"), "clicking the last item should open its menu");
});

test("navigation breadcrumb separates label navigation from menu selection", async () => {
  let navigated = 0;
  let selected = 0;
  const navigationItems = [
    { key: "alpha", title: "Project alpha", action: () => {} },
    { key: "beta", title: "Project beta", action: () => selected++ },
  ];

  await environment.render(
    React.createElement(BreadcrumbNavigationDropdown, {
      selectedItemKey: "alpha",
      navigationItems,
      handleOnClick: () => navigated++,
    })
  );

  assertNoNestedButtons(environment.container);
  assert.equal(environment.container.querySelectorAll("button").length, 2);
  await act(async () => environment.container.querySelector("button")?.click());
  assert.equal(navigated, 1);
  await act(async () => environment.container.querySelector("button[aria-label='Open breadcrumb menu']")?.click());
  const option = [...document.querySelectorAll("button")].find((button) => button.textContent.includes("Project beta"));
  assert.ok(option, "the chevron trigger should display menu items");
  await act(async () => option.click());
  assert.equal(selected, 1);
});
