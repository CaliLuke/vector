import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { useExampleReactions, toggleExampleReaction } =
  await import("../../packages/propel/src/emoji-reaction/reaction-examples.ts");

await test("reaction examples retain additions made in one event and toggle only the chosen reaction", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const initial = Object.freeze([
    Object.freeze({ emoji: "👍", count: 3, reacted: false, users: ["Alice", "Bob", "Charlie"] }),
    Object.freeze({ emoji: "❤️", count: 2, reacted: true, users: ["You", "David"] }),
  ]);
  function Harness() {
    const { reactions, addReaction, toggleReaction } = useExampleReactions(initial);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("output", null, JSON.stringify(reactions)),
      React.createElement(
        "button",
        {
          onClick: () => {
            addReaction("🎉");
            addReaction("👍");
          },
        },
        "Add reactions"
      ),
      React.createElement("button", { onClick: () => toggleReaction("👍") }, "Toggle thumbs up")
    );
  }
  await env.render(React.createElement(Harness));
  await act(async () => env.container.querySelector("button").click());
  const read = () => JSON.parse(env.container.querySelector("output").textContent);
  assert.deepEqual(read(), [
    { emoji: "👍", count: 4, reacted: true, users: ["Alice", "Bob", "Charlie"] },
    { emoji: "❤️", count: 2, reacted: true, users: ["You", "David"] },
    { emoji: "🎉", count: 1, reacted: true, users: ["You"] },
  ]);
  await act(async () => env.container.querySelectorAll("button")[1].click());
  assert.deepEqual(read()[0], { emoji: "👍", count: 3, reacted: false, users: ["Alice", "Bob", "Charlie"] });
  await act(async () => env.container.querySelectorAll("button")[1].click());
  assert.equal(read()[0].count, 4);
  assert.equal(read()[0].reacted, true);
  assert.equal(read()[1].count, 2);
  assert.equal(read()[2].count, 1);
  assert.equal(initial[0].count, 3);
});

await test("toggling an absent example reaction leaves the existing reactions unchanged", () => {
  const reactions = Object.freeze([Object.freeze({ emoji: "👍", count: 3 })]);
  assert.deepEqual(toggleExampleReaction(reactions, "🎉"), reactions);
});
