import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerSpaceModuleHooks();
const state = { toasts: [], React };
globalThis.publicClipboardFixture = state;
const stubs = {
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "lucide-react": "export const LinkIcon=()=>null,MoveRight=()=>null;",
  "@plane/propel/icons":
    "const Icon=()=>null;export const LinkIcon=Icon,CenterPanelIcon=Icon,FullScreenPanelIcon=Icon,SidePanelIcon=Icon,StatePropertyIcon=Icon,StateGroupIcon=Icon,PriorityPropertyIcon=Icon,DueDatePropertyIcon=Icon,PriorityIcon=Icon;",
  "@plane/propel/toast":
    'export const TOAST_TYPE={SUCCESS:"success",INFO:"info",ERROR:"error"};export const setToast=t=>globalThis.publicClipboardFixture.toasts.push(t);',
  "@headlessui/react":
    "export const Listbox=()=>null;Listbox.Button=()=>null;Listbox.Options=()=>null;Listbox.Option=()=>null;export const Transition=()=>null;",
  "@plane/hooks": "export const useRecordCollection=()=>({data:[]});",
  "@tanstack/react-query": 'export const useQuery=()=>({data:{project_details:{identifier:"DEMO"}}});',
  "react-router": 'export const useParams=()=>({anchor:"public"});',
  "@/queries/lookups": "export const statesOptions=()=>({});",
  "@/queries/publishing": "export const publishSettingsOptions=()=>({});",
  "@/helpers/date-time.helper": 'export const renderFormattedDate=()=>"date";',
  "@/helpers/issue.helper": "export const shouldHighlightIssueDueDate=()=>false;",
};
const stringSource = new URL("../../packages/utils/src/string.ts", import.meta.url).href;
registerHooks({
  resolve(specifier, context, next) {
    if (specifier === "@plane/utils/string") return next(stringSource, context);
    return resolvePackageStub(specifier, stubs, context, next);
  },
});
const { PeekOverviewHeader } = await import("../../apps/space/components/issues/peek-overview/header.tsx");
const { PeekOverviewIssueProperties } =
  await import("../../apps/space/components/issues/peek-overview/issue-properties.tsx");
const cases = [
  { name: "header", Component: PeekOverviewHeader, props: { peekMode: "modal", handleClose() {}, setPeekMode() {} } },
  {
    name: "properties",
    Component: PeekOverviewIssueProperties,
    props: { mode: "full", issueDetails: { id: "issue", sequence_id: 3 } },
  },
];
for (const { name, Component, props } of cases)
  for (const backend of ["modern", "legacy"])
    void test(`public ${name} reports ${backend} copy denial and permits retry`, async (t) => {
      const env = await createReactDomTestEnvironment();
      t.after(() => env.unmount());
      state.toasts.length = 0;
      const copies = [];
      let accept = false;
      if (backend === "modern")
        Object.defineProperty(navigator, "clipboard", {
          configurable: true,
          value: {
            writeText: async (text) => {
              copies.push(text);
              if (!accept) throw new Error("clipboard denied");
            },
          },
        });
      else {
        Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
        document.execCommand = (command) => {
          copies.push([command, document.activeElement.value]);
          return accept;
        };
      }
      await env.render(React.createElement(Component, props));
      const button = env.container.querySelector("button");
      assert.ok(button);
      await act(async () => button.click());
      assert.equal(state.toasts.length, 1);
      assert.equal(state.toasts[0].type, "error");
      accept = true;
      await act(async () => button.click());
      assert.equal(state.toasts.length, 2);
      assert.notEqual(state.toasts[1].type, "error");
      assert.equal(copies.length, 2);
      assert.equal(document.querySelectorAll("textarea").length, 0);
    });

for (const permission of ["unsupported", "prompt"]) {
  void test(`public copy stays available when clipboard permission is ${permission}`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    state.toasts.length = 0;
    Object.defineProperty(navigator, "permissions", {
      configurable: true,
      value: {
        query: async () => {
          if (permission === "unsupported") throw new TypeError("Unsupported permission name");
          return { state: "prompt" };
        },
      },
    });
    const writes = [];
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: async (text) => {
          writes.push(text);
        },
      },
    });
    await env.render(
      React.createElement(PeekOverviewHeader, { peekMode: "modal", handleClose() {}, setPeekMode() {} })
    );
    const button = env.container.querySelector("button");
    assert.ok(button, "user-initiated copying must remain available");
    await act(async () => button.click());
    assert.deepEqual(writes, [window.location.href]);
    assert.equal(state.toasts.at(-1).type, "success");
  });
}
