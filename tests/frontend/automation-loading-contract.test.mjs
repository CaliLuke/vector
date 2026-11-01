import assert from "node:assert/strict";
import test from "node:test";
import { React, act, registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";

globalThis.automationLoadingReact = React;
globalThis.automationLoadingState = { project: undefined, isAdmin: true };
const ui = `
  export {Loader} from ${JSON.stringify(new URL("../../packages/ui/src/loader.tsx", import.meta.url).href)};
  export {ToggleSwitch} from ${JSON.stringify(new URL("../../packages/ui/src/button/toggle-switch.tsx", import.meta.url).href)};
  export const CustomSelect=Object.assign(()=>null,{Option:()=>null});
  export const CustomSearchSelect=()=>null;
`;
const stubs = {
  "@plane/ui": ui,
  "@tanstack/react-router": "export const useParams=()=>({workspaceSlug:'acme'});",
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@/hooks/use-projects": "export const useRouteProject=()=>({project:globalThis.automationLoadingState.project});",
  "@/hooks/use-permissions":
    "export const useProjectPermissions=(_workspace,id)=>({can:()=>Boolean(id)&&globalThis.automationLoadingState.isAdmin});",
  "@/hooks/use-project-states":
    "export const useProjectStates=(_workspace,id)=>({projectStates:id?[{id:'cancelled',name:'Cancelled',group:'cancelled'}]:[]});",
  "@/components/automation": "export const SelectMonthModal=()=>null;",
};
registerPermissionUiStubs("/automation/auto-archive-automation.tsx", stubs);
registerPermissionUiStubs("/automation/auto-close-automation.tsx", stubs);
registerPermissionUiStubs("/automation/automation-options-loader.tsx", { "@plane/ui": ui });

async function registerAutomationContract(name, path, exported, enable, disable) {
  await test(`${name} automation waits for its project and retains its toggle payload and permission gate`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const Component = (await import(path))[exported];
    const state = (globalThis.automationLoadingState = { project: undefined, isAdmin: true });
    const writes = [];
    const render = () =>
      env.render(
        React.createElement(Component, {
          handleChange: async (data) => {
            writes.push(data);
            state.project = { ...state.project, ...data };
          },
        })
      );
    await render();
    const toggle = () => env.container.querySelector('[role="switch"]');
    assert.ok(env.container.querySelector('[aria-busy="true"]'));
    assert.equal(toggle().disabled, true);
    await act(async () => toggle().click());
    assert.deepEqual(writes, []);
    state.project = { id: "p1", archive_in: 0, close_in: 0 };
    await render();
    assert.equal(Boolean(env.container.querySelector('[aria-busy="true"]')), false);
    assert.equal(toggle().disabled, false);
    await act(async () => toggle().click());
    assert.deepEqual(writes, [enable]);
    await render();
    assert.equal(toggle().getAttribute("aria-checked"), "true");
    await act(async () => toggle().click());
    assert.deepEqual(writes, [enable, disable]);
    state.isAdmin = false;
    await render();
    assert.equal(toggle().disabled, true);
    await act(async () => toggle().click());
    assert.deepEqual(writes, [enable, disable]);
  });
}

await registerAutomationContract(
  "Archive",
  "../../apps/web/core/components/automation/auto-archive-automation.tsx",
  "AutoArchiveAutomation",
  { archive_in: 1 },
  { archive_in: 0 }
);
await registerAutomationContract(
  "Close",
  "../../apps/web/core/components/automation/auto-close-automation.tsx",
  "AutoCloseAutomation",
  { close_in: 1, default_state: "cancelled" },
  { close_in: 0, default_state: null }
);
