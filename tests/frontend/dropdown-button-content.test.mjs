import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@/hooks/use-platform-os") {
      return {
        shortCircuit: true,
        url: "data:text/javascript,export const usePlatformOS = () => ({ isMobile: false });",
      };
    }
    return nextResolve(specifier, context);
  },
});

for (const variant of [
  "border-with-text",
  "border-without-text",
  "background-with-text",
  "background-without-text",
  "transparent-with-text",
  "transparent-without-text",
]) {
  test(`${variant} dropdown content leaves one focusable trigger and respects its disabled state`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const errors = [];
    t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
    const { DropdownButton } = await import("../../apps/web/core/components/dropdowns/buttons.tsx");
    let openings = 0;
    const content = React.createElement(
      DropdownButton,
      {
        variant,
        isActive: false,
        tooltipHeading: "State",
        tooltipContent: "Backlog",
        showTooltip: false,
      },
      "Backlog"
    );
    const renderTrigger = (disabled) =>
      environment.render(
        React.createElement("button", { type: "button", disabled, onClick: () => openings++ }, content)
      );

    await renderTrigger(false);
    assert.equal(environment.container.querySelectorAll("button").length, 1);
    const trigger = environment.container.querySelector("button");
    trigger.focus();
    assert.equal(document.activeElement, trigger);
    await act(async () => trigger.firstElementChild.click());
    assert.equal(openings, 1);

    await renderTrigger(true);
    await act(async () => trigger.firstElementChild.click());
    assert.equal(openings, 1);
    assert.deepEqual(errors, []);
  });
}
