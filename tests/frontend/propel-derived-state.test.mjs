import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { React, act } from "./helpers/react.mjs";
registerWebModuleHooks();
const { AnimatedCounter } = await import("../../packages/propel/src/animated-counter/animated-counter.tsx");
const { IconRoot } = await import("../../packages/propel/src/emoji-icon-picker/icon/icon-root.tsx");

void test("counter settles the latest value even when an animation reverses before completion", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  t.mock.timers.enable({ apis: ["setTimeout"] });
  await env.render(React.createElement(AnimatedCounter, { count: 1 }));
  assert.equal(env.container.textContent, "1");
  await env.render(React.createElement(AnimatedCounter, { count: 2 }));
  assert.equal(env.container.querySelector(".animate-slide-in-from-bottom").textContent, "2");
  await act(async () => t.mock.timers.tick(100));
  await env.render(React.createElement(AnimatedCounter, { count: 1 }));
  assert.equal(env.container.querySelector(".animate-slide-in-from-top").textContent, "1");
  await act(async () => t.mock.timers.tick(249));
  assert.equal(env.container.querySelectorAll("span").length, 2);
  await act(async () => t.mock.timers.tick(1));
  assert.equal(env.container.querySelectorAll("span").length, 1);
  assert.equal(env.container.textContent, "1");
});

void test("icon selection uses the replacement default color while preserving the search", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const changes = [];
  const render = (defaultColor) =>
    env.render(
      React.createElement(IconRoot, {
        defaultColor,
        iconType: "lucide",
        onChange: (value) => changes.push(value),
      })
    );
  await render("#95999f");
  const search = env.container.querySelector('input[placeholder="Search"]');
  await changeFormControl(search, "activity");
  await render("#112233");
  assert.equal(env.container.querySelector('input[placeholder="Search"]').value, "activity");
  assert.equal(env.container.querySelector('input[type="text"]').value, "112233");
  const icon = env.container.querySelector('button[aria-label="Activity"]');
  assert.ok(icon);
  await act(async () => icon.click());
  assert.equal(changes.at(-1).color, "#112233");
});
