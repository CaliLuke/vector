import { registerDropdownSourceModules } from "./helpers/dropdown-source-modules.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";

const customMenuUrl = new URL("../../packages/ui/src/dropdowns/custom-menu.tsx", import.meta.url).href;
const buttonUrl = new URL("../../packages/propel/src/button/index.ts", import.meta.url).href;
const substitutes = {
  "@plane/ui": `export { CustomMenu } from ${JSON.stringify(customMenuUrl)};`,
  "@plane/propel/button": `export { Button } from ${JSON.stringify(buttonUrl)};`,
  "@plane/propel/icons":
    "export const CheckIcon = () => null; export const ChevronDownIcon = () => null; export const ChevronRightIcon = () => null; export const ChevronUpIcon = () => null;",
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
  "@plane/types":
    "export const EUserProjectRoles = { ADMIN: 20, MEMBER: 15, GUEST: 5 }; export const EUserWorkspaceRoles = { ADMIN: 20, MEMBER: 15, GUEST: 5 };",
  "@plane/utils/common":
    "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');",
};

async function flushAnimationFrames() {
  await act(async () => {
    await new Promise((resolve) => {
      let frames = 0;
      const flushNextFrame = () => {
        frames += 1;
        if (frames === 6) resolve();
        else requestAnimationFrame(flushNextFrame);
      };
      requestAnimationFrame(flushNextFrame);
    });
  });
}

registerDropdownSourceModules("custom-menu.tsx", substitutes);

test("member role filter uses one focusable trigger and Escape returns focus", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { MemberListFiltersDropdown } =
    await import("../../apps/web/core/components/project/dropdowns/filters/member-list.tsx");
  let appliedRole = null;

  await environment.render(
    React.createElement(MemberListFiltersDropdown, {
      appliedFilters: [],
      handleUpdate: (role) => (appliedRole = role),
      memberType: "workspace",
    })
  );

  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  assert.equal(environment.container.querySelectorAll("button").length, 1, "trigger should not nest buttons");
  assert.equal(trigger.textContent.trim(), "Filters");
  await act(() => trigger.focus());
  await flushAnimationFrames();
  await act(async () => trigger.click());
  await flushAnimationFrames();
  const role = [...document.querySelectorAll("button")].find((button) => button.textContent.includes("Admin"));
  assert.ok(role, "opening the menu should show role filters");
  assert.ok(document.querySelector('[role="menu"]'), "the real CustomMenu should expose its menu");
  await act(() => role.click());
  await flushAnimationFrames();
  assert.equal(appliedRole, "20", "selecting Admin should apply the corresponding workspace role filter");
  assert.ok(document.querySelector('[role="menu"]'), "the filter menu should stay open after applying a role");

  await act(async () => {
    const escapeTarget = document.activeElement instanceof HTMLElement ? document.activeElement : role;
    escapeTarget.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  await flushAnimationFrames();
  assert.equal(document.querySelector('[role="menu"]'), null, "Escape should close the menu");
  const focusDeadline = performance.now() + 200;
  await act(
    async () =>
      new Promise((resolve) => {
        const checkFocus = () => {
          if (document.activeElement === trigger || performance.now() >= focusDeadline) resolve();
          else setTimeout(checkFocus, 10);
        };
        checkFocus();
      })
  );
  assert.ok(document.activeElement === trigger, "Escape should restore focus to the filter trigger");
  assert.equal(appliedRole, "20", "Escape should leave the selected role unchanged");
  assert.deepEqual(errors, [], "render should not emit DOM nesting warnings");
});
