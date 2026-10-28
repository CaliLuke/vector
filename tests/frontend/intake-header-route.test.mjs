import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createRouterFixture } from "./helpers/router.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const stubs = {
  "@/hooks/use-projects": "export const useProjectDetail = () => ({isPending: false});",
  "@/hooks/use-permissions": "export const useProjectPermissions = () => ({can: () => false});",
  "@/components/inbox/modals/create-modal": "export const InboxIssueCreateModalRoot = () => null;",
  "@/components/breadcrumbs/common": "export const CommonProjectBreadcrumbs = () => null;",
  "@plane/i18n": "export const useTranslation = () => ({t: key => key});",
};
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (Object.hasOwn(stubs, specifier))
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    return nextResolve(specifier, context);
  },
});
const { ProjectInboxHeader } = await import("../../apps/web/core/components/projects/settings/intake/header.tsx");

await test("intake header reads the project from its intake route without a settings match", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const fixture = createRouterFixture("/acme/projects/project-id/intake/");
  await fixture.render(environment, React.createElement(ProjectInboxHeader));
  const link = [...environment.container.querySelectorAll("a")].find((element) => element.textContent === "Intake");
  assert.ok(link, "intake header renders outside project settings");
  assert.equal(link.getAttribute("href"), "/acme/projects/project-id/intake/");
});
