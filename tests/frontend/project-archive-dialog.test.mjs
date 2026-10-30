import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClientProvider } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.projectArchiveDialogReact = React;
registerHooks({
  resolve(specifier, context, next) {
    if (context.parentURL?.endsWith("/project/archive-restore-modal.tsx")) {
      const substitutes = {
        "@plane/ui": `const React=globalThis.projectArchiveDialogReact;
          export const EModalPosition={CENTER:'center'}, EModalWidth={LG:'lg'};
          export const ModalCore=({isOpen,children})=>isOpen?React.createElement('div',{role:'dialog'},children):null;`,
        "@plane/propel/button": `const React=globalThis.projectArchiveDialogReact;
          export const Button=({children,onClick,loading})=>React.createElement('button',{onClick,disabled:loading},children);`,
        "@plane/propel/toast":
          "export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};export const setToast=()=>{};",
      };
      if (Object.hasOwn(substitutes, specifier))
        return { shortCircuit: true, url: "data:text/javascript," + encodeURIComponent(substitutes[specifier]) };
    }
    return next(specifier, context);
  },
});
const { ArchiveRestoreProjectModal } = await import("../../apps/web/core/components/project/archive-restore-modal.tsx");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const { ProjectArchiveService } = await import("../../apps/web/core/services/project/project-archive.service.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");

await test("an archived list record can be restored without an active-project detail read", async (t) => {
  const { env, client } = await createQueryDomFixture(t);
  const router = createRouterFixture("/acme/projects/archives/");
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole: 20,
    projectRoles: { "project-1": 20 },
  });
  const detail = t.mock.method(ProjectService.prototype, "getProject", async () => {
    throw new Error("Archived projects are absent from active detail reads");
  });
  const restore = t.mock.method(ProjectArchiveService.prototype, "restoreProject", async () => {});
  let closed = 0;
  await router.render(
    env,
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(ArchiveRestoreProjectModal, {
        workspaceSlug: "acme",
        project: { id: "project-1", name: "Archived example" },
        isOpen: true,
        archive: false,
        onClose: () => {
          closed += 1;
        },
      })
    )
  );
  assert.match(env.container.querySelector('[role="dialog"]')?.textContent ?? "", /Restore Archived example/);
  const button = [...env.container.querySelectorAll("button")].find((item) => item.textContent === "Restore");
  await act(async () => button.click());
  assert.equal(detail.mock.callCount(), 0);
  assert.deepEqual(
    restore.mock.calls.map((call) => call.arguments),
    [["acme", "project-1"]]
  );
  assert.equal(closed, 1);
  assert.equal(router.router.state.location.pathname, "/acme/projects/");
});
