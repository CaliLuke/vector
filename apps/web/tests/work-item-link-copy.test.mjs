import assert from "node:assert/strict";
import { setup, React, identity } from "./helpers/issue-lifecycle-fixture.mjs";
import { registerWorkItemCopyTests } from "./helpers/work-item-copy-contract.mjs";
const { ProjectIssueQuickActions } =
  await import("../core/components/issues/issue-layouts/quick-action-dropdowns/project-issue.tsx");
for (const mode of ["detail", "peek", "project"]) {
  registerWorkItemCopyTests(
    `${mode} copy control`,
    async (t) => {
      const view = await setup(t, mode === "project" ? "detail" : mode);
      if (mode === "project")
        await view.renderElement(
          React.createElement(ProjectIssueQuickActions, {
            issue: {
              id: "item",
              project_id: "p1",
              name: "Original",
              sequence_id: 1,
              created_by: "alice",
              state_id: "completed",
            },
          })
        );
      t.after(() => assert.deepEqual(view.state.unhandled, []));
      return {
        ...view,
        path: "/acme/browse/P1-1/",
        copyButton: () => {
          const button =
            mode === "project"
              ? view.action("copy-link")
              : view.environment.container.querySelector('[aria-label="common.actions.copy_link"]');
          assert.ok(button, "native copy control mounted");
          return button;
        },
        toasts: () => view.state.toasts,
        unmount: () => view.renderElement(null),
      };
    },
    identity
  );
}
