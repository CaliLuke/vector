import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";

registerWebTestModules({}, (specifier, context, nextResolve) => {
  if (specifier === "@/hooks/use-platform-os") {
    return {
      shortCircuit: true,
      url: "data:text/javascript,export const usePlatformOS = () => ({ isMobile: false });",
    };
  }
  return nextResolve(specifier, context);
});

for (const variant of [
  "border-with-text",
  "border-without-text",
  "background-with-text",
  "background-without-text",
  "transparent-with-text",
  "transparent-without-text",
]) {
  // Register eagerly; node:test owns completion and reports failures before suite cleanup.
  void test(`${variant} dropdown content leaves one focusable trigger and respects its disabled state`, async (t) => {
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

void test("custom dropdown content retains one native, forwarded trigger and respects disabled state", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { DropdownTrigger } = await import("../../apps/web/core/components/dropdowns/buttons.tsx");
  const ref = React.createRef();
  let clicks = 0;
  const render = (disabled, custom = true) =>
    env.render(
      React.createElement(
        DropdownTrigger,
        {
          appearance: {
            button: custom ? React.createElement("span", null, "Custom label") : undefined,
            disabled,
            buttonVariant: "border-with-text",
          },
          ref,
          isActive: false,
          tooltipHeading: "State",
          onClick: () => clicks++,
        },
        "Fallback label"
      )
    );
  await render(false);
  assert.equal(env.container.querySelectorAll("button").length, 1);
  assert.equal(ref.current, env.container.querySelector("button"));
  assert.equal(ref.current.type, "button");
  assert.equal(ref.current.textContent, "Custom label");
  await act(async () => ref.current.click());
  assert.equal(clicks, 1);
  await render(true);
  await act(async () => ref.current.click());
  assert.equal(clicks, 1);
  assert.equal(ref.current.disabled, true);
  const nativeTrigger = ref.current;
  await render(true, false);
  assert.equal(ref.current, nativeTrigger);
  assert.equal(ref.current.textContent, "Fallback label");
  await act(async () => ref.current.firstElementChild.click());
  assert.equal(clicks, 1, "fallback content retains the disabled native trigger");
});
