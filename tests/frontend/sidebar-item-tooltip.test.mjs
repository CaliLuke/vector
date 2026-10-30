import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

function testSidebarTooltip(variant) {
  return test(`sidebar ${variant} forwards tooltip focus handlers and accessible descriptions`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const { createRouterFixture } = await import("./helpers/router.mjs");
    const { AppSidebarItem } = await import("../../apps/web/core/components/sidebar/sidebar-item.tsx");
    const { Tooltip } = await import("../../packages/propel/src/tooltip/root.tsx");
    const router = createRouterFixture("/workspace/");
    const ref = React.createRef();
    let focused = false;
    await router.render(
      environment,
      React.createElement(
        Tooltip,
        { tooltipContent: "Navigation details", openDelay: 0 },
        React.createElement(AppSidebarItem, {
          variant,
          item: { href: "/workspace/", label: "Navigation" },
          ref,
          tabIndex: 0,
          "aria-label": "Navigation target",
          "aria-describedby": "navigation-description",
          className: "consumer-style",
          onFocus: () => {
            focused = true;
          },
        })
      )
    );

    await act(async () => {
      ref.current.focus();
    });

    assert.equal(focused, true);
    assert.match(document.body.textContent, /Navigation details/);
    assert.equal(ref.current.getAttribute("aria-describedby"), "navigation-description");
    assert.equal(ref.current.getAttribute("aria-label"), "Navigation target");
    assert.ok(ref.current.classList.contains("consumer-style"));
  });
}

await testSidebarTooltip("link");
await testSidebarTooltip("button");
await testSidebarTooltip("content");

await test("sidebar button preserves item actions and consumer cancellation", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { AppSidebarItem } = await import("../../apps/web/core/components/sidebar/sidebar-item.tsx");
  let actions = 0;
  let clicks = 0;
  const item = {
    label: "Action",
    onClick: () => {
      actions++;
    },
  };
  const render = (onClick) =>
    environment.render(React.createElement(AppSidebarItem, { variant: "button", item, onClick }));
  await render(() => {
    clicks++;
  });

  await act(async () => environment.container.querySelector("button").click());
  assert.equal(actions, 1);
  assert.equal(clicks, 1);

  await render((event) => event.preventDefault());
  await act(async () => environment.container.querySelector("button").click());
  assert.equal(actions, 1);
});
