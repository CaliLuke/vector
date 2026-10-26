import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const stubs = {
  "@/hooks/use-platform-os": `export const usePlatformOS = () => ({ isMobile: false });`,
  "@plane/propel/tooltip": `export const Tooltip = ({children}) => children;`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

await test("selected module chips remain non-interactive content inside the picker trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ModuleButtonContent } = await import("../../apps/web/core/components/dropdowns/module/button-content.tsx");
  let opened = 0;
  await environment.render(
    React.createElement(
      "button",
      { type: "button", onClick: () => opened++ },
      React.createElement(ModuleButtonContent, {
        value: ["module"],
        modules: [{ id: "module", name: "Selected module" }],
        showCount: false,
        hideIcon: false,
        hideText: false,
        dropdownArrow: true,
        dropdownArrowClassName: "",
      })
    )
  );
  const trigger = environment.container.querySelector("button");
  assert.equal(trigger.textContent, "Selected module");
  assert.equal(trigger.querySelector("button, [role=button], input, a"), null);
  await act(async () => trigger.click());
  assert.equal(opened, 1);
});
