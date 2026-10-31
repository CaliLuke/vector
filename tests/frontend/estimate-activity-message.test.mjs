import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const estimateURL = new URL(
  "../../apps/web/core/components/issues/issue-detail/issue-activity/activity/actions/estimate.tsx",
  import.meta.url
).href;
const unrelated = [
  "Default",
  "Name",
  "Description",
  "State",
  "Assignee",
  "Priority",
  "Parent",
  "Relation",
  "Date",
  "Cycle",
  "Module",
  "Label",
  "Link",
  "Attachment",
  "ArchivedAt",
  "Inbox",
]
  .map((name) => `empty as Issue${name}Activity`)
  .join(",");
registerHooks({
  resolve(specifier, context, nextResolve) {
    const stubs = {
      "@/components/relations": "export const useTimeLineRelationOptions = () => ({});",
    };
    if (context.parentURL?.endsWith("/activity/activity-list.tsx")) {
      stubs["./actions"] =
        `export { IssueEstimateActivity } from ${JSON.stringify(estimateURL)}; const empty=()=>null; export {${unrelated}};`;
      stubs["./actions/property"] = "export const IssuePropertyActivity=()=>null;";
    }
    if (context.parentURL === estimateURL) {
      stubs["./"] = "export const IssueActivityBlockComponent=({children})=>children; export const IssueLink=()=>null;";
    }
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});
const { IssueActivityItem } =
  await import("../../apps/web/core/components/issues/issue-detail/issue-activity/activity/activity-list.tsx");

await Promise.all(
  ["estimate_points", "estimate_categories", "estimate_time", "estimate_point"].map((field) =>
    test(`work-item history displays ${field} and its retained removal label`, async (t) => {
      const env = await createReactDomTestEnvironment();
      t.after(() => env.unmount());
      await env.render(
        React.createElement(IssueActivityItem, { activity: { field, new_value: "2h", old_value: null } })
      );
      assert.equal(document.body.textContent, "set the estimate point to 2h.");
      await env.render(
        React.createElement(IssueActivityItem, { activity: { field, new_value: null, old_value: "<b>old</b>" } })
      );
      assert.equal(document.body.textContent, "removed the estimate point <b>old</b>.");
      assert.equal(document.querySelector("b"), null);
    })
  )
);
