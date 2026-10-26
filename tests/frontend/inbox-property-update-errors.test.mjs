import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

const context = "globalThis.inboxPropertyErrorTest";
globalThis.inboxPropertyErrorTest = { React };
const picker = (name, value) =>
  `export const ${name}=({onChange,disabled})=>${context}.React.createElement('button',{disabled,onClick:()=>onChange(${value})},'${name}');`;
registerWebTestModules({
  "@tanstack/react-router": "export const useRouter=()=>({});",
  "@plane/propel/icons": ["State", "Members", "Priority", "DueDate", "Label", "Duplicate"]
    .map((name) => `export const ${name}PropertyIcon=()=>null;`)
    .join(""),
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@plane/ui": "export const ControlLink=()=>null;",
  "@/hooks/use-projects": "export const useProjectDetail=()=>({});",
  "@/components/dropdowns/date": picker("DateDropdown", "new Date(2026, 9, 27)"),
  "@/components/dropdowns/member/dropdown": picker("MemberDropdown", "['alice']"),
  "@/components/dropdowns/priority": picker("PriorityDropdown", "'high'"),
  "@/components/dropdowns/intake-state/dropdown": "export const IntakeStateDropdown=()=>null;",
  "@/components/dropdowns/state/dropdown": "export const StateDropdown=()=>null;",
  "@/components/issues/issue-detail/label": "export const IssueLabel=()=>null;",
});
const { InboxIssueContentProperties } =
  await import("../../apps/web/core/components/inbox/content/issue-properties.tsx");

await test("failed inbox property writes stay handled at picker event boundaries and remain retryable", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const writes = [];
  let fail = true;
  await env.render(
    React.createElement(InboxIssueContentProperties, {
      workspaceSlug: "acme",
      projectId: "project",
      issue: { id: "issue", assignee_ids: [] },
      isEditable: true,
      isIntakeAccepted: false,
      issueOperations: {
        update: async (...args) => {
          writes.push(args);
          if (fail) throw new Error("The operation already reported this failure.");
        },
      },
    })
  );
  const buttons = [...env.container.querySelectorAll("button")];
  assert.equal(buttons.length, 3);
  await act(async () => {
    for (const button of buttons) button.click();
    await new Promise(setImmediate);
  });
  assert.deepEqual(writes, [
    ["acme", "project", "issue", { assignee_ids: ["alice"] }],
    ["acme", "project", "issue", { priority: "high" }],
    ["acme", "project", "issue", { target_date: "2026-10-27" }],
  ]);
  fail = false;
  await act(async () => {
    buttons[1].click();
    await new Promise(setImmediate);
  });
  assert.deepEqual(writes[3], writes[1]);
});
