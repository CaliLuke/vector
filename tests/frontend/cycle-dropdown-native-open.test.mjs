import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const stubs = {
  "@/hooks/use-platform-os": `export const usePlatformOS = () => ({ isMobile: false });`,
  "@/components/dropdowns/buttons": `export const DropdownButton = ({children}) => children;`,
  "@/components/dropdowns/constants": `export const BUTTON_VARIANTS_WITH_TEXT = ["transparent-with-text"];`,
  "@plane/i18n": `export const useTranslation = () => ({ t: key => key });`,
  "@plane/propel/icons": `export const CycleIcon = () => null; export const ChevronDownIcon = () => null;
    export const CheckIcon = () => null; export const CycleGroupIcon = () => null; export const ModuleIcon = () => null;
    export const SearchIcon = () => null;`,
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/ui") {
      return nextResolve(
        pathToFileURL(new URL("../../packages/ui/src/dropdowns/combo-box.tsx", import.meta.url).pathname).href,
        context
      );
    }
    if (specifier === "@headlessui/react") {
      return nextResolve(specifier, {
        ...context,
        parentURL: pathToFileURL(new URL("../../apps/web/index.js", import.meta.url).pathname).href,
      });
    }
    if (Object.hasOwn(stubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

function setPositioningGeometry(trigger) {
  Object.defineProperty(document.documentElement, "clientWidth", { configurable: true, value: 1200 });
  Object.defineProperty(document.documentElement, "clientHeight", { configurable: true, value: 800 });
  trigger.getBoundingClientRect = () => ({
    x: 400,
    y: 100,
    left: 400,
    top: 100,
    right: 600,
    bottom: 132,
    width: 200,
    height: 32,
    toJSON() {},
  });
}

await test("a cycle picker shares one native catalogue read across repeated opens", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const environment = fixture.environment;
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(fixture.client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole: 15,
    projectRoles: { "project-1": 20 },
  });
  const fetches = [];
  t.mock.method(CycleService.prototype, "getWithParams", async (workspaceSlug, projectId) => {
    fetches.push([workspaceSlug, projectId]);
    return [];
  });
  const { CycleDropdown } = await import("../../apps/web/core/components/dropdowns/cycle/index.tsx");

  await fixture.render(
    React.createElement(CycleDropdown, {
      buttonVariant: "transparent-with-text",
      onChange() {},
      projectId: "project-1",
      value: null,
      canRemoveCycle: false,
    })
  );
  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  setPositioningGeometry(trigger);

  await clickWithPointer(trigger);
  await settleQuery();
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.deepEqual(fetches, [["acme", "project-1"]]);
  const options = environment.container.querySelector('[role="listbox"]');
  assert.ok(options);
  assert.equal(options.style.position, "absolute");
  assert.equal(options.getAttribute("data-popper-placement"), "bottom-start");
  assert.equal(options.style.transform, "translate(400px, 132px)");

  await clickWithPointer(trigger);
  await settleQuery();
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  await clickWithPointer(trigger);
  await settleQuery();
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.deepEqual(fetches, [["acme", "project-1"]]);
});

await test("the module picker positions its public options element with Popper", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const environment = fixture.environment;
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(fixture.client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole: 15,
    projectRoles: { "project-1": 20 },
  });
  const fetches = [];
  t.mock.method(ModuleService.prototype, "getModules", async (workspaceSlug, projectId) => {
    fetches.push([workspaceSlug, projectId]);
    return [];
  });
  const { ModuleDropdown } = await import("../../apps/web/core/components/dropdowns/module/dropdown.tsx");

  await fixture.render(
    React.createElement(ModuleDropdown, {
      buttonVariant: "transparent-with-text",
      onChange() {},
      projectId: "project-1",
      value: null,
      multiple: false,
    })
  );
  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  setPositioningGeometry(trigger);

  await clickWithPointer(trigger);
  await settleQuery();
  const options = environment.container.querySelector('[role="listbox"]');
  assert.ok(options);
  assert.equal(options.getAttribute("data-popper-placement"), "bottom-start");
  assert.equal(options.style.transform, "translate(400px, 132px)");
  assert.deepEqual(fetches, [["acme", "project-1"]]);
});
