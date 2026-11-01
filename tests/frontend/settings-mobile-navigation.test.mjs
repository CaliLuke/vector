import assert from "node:assert/strict";
import test from "node:test";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
registerWebTestModules({ "@plane/i18n": "export const useTranslation=()=>({t:key=>key});" });
const { SettingsMobileNav } = await import("../../apps/web/core/components/settings/mobile/nav.tsx");
const { getWorkspaceActivePath, getProjectActivePath } =
  await import("../../apps/web/core/components/settings/helper.ts");

void test("settings navigation resolves workspace and project sections without matching unrelated routes", () => {
  assert.equal(getWorkspaceActivePath("/acme/settings"), "workspace_settings.settings.general.title");
  assert.equal(getProjectActivePath("/acme/settings/projects/project/members/"), "common.members");
  assert.equal(getProjectActivePath("/acme/settings/projects/project/"), "common.general");
  assert.equal(getWorkspaceActivePath("/acme/projects/project/"), null);
  assert.equal(getProjectActivePath("/acme/projects/project/"), null);
  assert.equal(getWorkspaceActivePath("/acme/settings/unknown/"), undefined);
  assert.equal(getProjectActivePath("/acme/settings/projects/project/unknown/"), undefined);
});

void test("open settings navigation retains its sidebar state when the active path updates", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let mounts = 0;
  function Sidebar() {
    React.useEffect(() => {
      mounts++;
    }, []);
    return React.createElement("input", { "aria-label": "Find settings", defaultValue: "" });
  }
  const render = (activePath) =>
    env.render(React.createElement(SettingsMobileNav, { activePath }, React.createElement(Sidebar)));
  await render("General");
  const trigger = env.container.querySelector('button[aria-label="Settings navigation"]');
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(env.container.querySelector("input"), null);
  await act(async () => trigger.click());
  const input = env.container.querySelector("input");
  input.value = "Members";
  input.focus();
  await render("Members");
  assert.equal(env.container.querySelector("input"), input);
  assert.equal(input.value, "Members");
  assert.equal(document.activeElement, input);
  assert.equal(mounts, 1);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  await act(async () => trigger.click());
  assert.equal(env.container.querySelector("input"), null);
});
