import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { act, React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks({ stubEmojiRoot: true });

// Register eagerly; node:test owns completion and reports failures before suite cleanup.
void test("reaction group keeps reaction buttons separate from its keyboard popover trigger", async () => {
  const environment = await createReactDomTestEnvironment();

  try {
    const { EmojiReactionPicker } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction-picker.tsx");
    const { EmojiReactionGroup } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction.tsx");
    const reactionClicks = [];
    const toggleCalls = [];
    const addCalls = [];

    function Harness() {
      const [isOpen, setIsOpen] = React.useState(false);
      return React.createElement(EmojiReactionPicker, {
        isOpen,
        handleToggle(value) {
          toggleCalls.push(value);
          setIsOpen(value);
        },
        buttonClassName: "picker-trigger",
        onChange() {},
        label: React.createElement(EmojiReactionGroup, {
          reactions: [{ emoji: "👍", count: 1 }],
          onReactionClick(emoji) {
            reactionClicks.push(emoji);
          },
          onAddReaction() {
            addCalls.push(true);
          },
        }),
      });
    }

    await environment.render(React.createElement(Harness));

    const reaction = environment.container.querySelector("button:not(.picker-trigger)");
    const addTrigger = environment.container.querySelector("button.picker-trigger");
    assert.ok(reaction);
    assert.ok(addTrigger);
    assert.equal(environment.container.querySelectorAll("button").length, 2);
    assert.equal(environment.container.querySelector("button button"), null);

    await act(async () => reaction.click());
    assert.deepEqual(reactionClicks, ["👍"]);
    assert.deepEqual(toggleCalls, []);

    async function checkKeyboardActivation(key) {
      await act(async () => {
        addTrigger.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
        // jsdom does not synthesize a native button's keyboard activation click.
        addTrigger.click();
      });
      assert.equal(toggleCalls.at(-1), true);
      assert.equal(addTrigger.getAttribute("aria-expanded"), "true", environment.container.innerHTML);
      assert.deepEqual(addCalls, []);
      await act(async () => addTrigger.click());
      assert.equal(addTrigger.getAttribute("aria-expanded"), "false");
      assert.equal(toggleCalls.at(-1), false);
    }
    await checkKeyboardActivation("Enter");
    await checkKeyboardActivation(" ");
  } finally {
    await environment.unmount();
  }
});

async function renderStaticPicker(environment, { disabled = false, showAddButton = true }) {
  const { EmojiReactionPicker } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction-picker.tsx");
  const { EmojiReactionGroup } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction.tsx");
  const reactionClicks = [];
  const toggleCalls = [];
  await environment.render(
    React.createElement(EmojiReactionPicker, {
      isOpen: false,
      disabled,
      handleToggle: (value) => toggleCalls.push(value),
      buttonClassName: "picker-trigger",
      onChange() {},
      label: React.createElement(EmojiReactionGroup, {
        reactions: [{ emoji: "👍", count: 1 }],
        onReactionClick: (emoji) => reactionClicks.push(emoji),
        showAddButton,
      }),
    })
  );
  return { reactionClicks, toggleCalls };
}

void test("a disabled reaction picker disables only its add trigger", async () => {
  const environment = await createReactDomTestEnvironment();

  try {
    const { reactionClicks, toggleCalls } = await renderStaticPicker(environment, { disabled: true });

    const addTrigger = environment.container.querySelector("button.picker-trigger");
    const reaction = environment.container.querySelector("button:not(.picker-trigger)");
    assert.ok(addTrigger?.disabled);
    assert.ok(reaction);
    await act(async () => {
      addTrigger.click();
      reaction.click();
    });

    assert.deepEqual(toggleCalls, []);
    assert.deepEqual(reactionClicks, ["👍"]);
    assert.equal(environment.container.querySelector("button button"), null);
  } finally {
    await environment.unmount();
  }
});

void test("a reaction group without an add button does not wrap reaction buttons in a trigger", async () => {
  const environment = await createReactDomTestEnvironment();

  try {
    const { reactionClicks, toggleCalls } = await renderStaticPicker(environment, { showAddButton: false });

    const reaction = environment.container.querySelector("button");
    assert.ok(reaction);
    assert.equal(environment.container.querySelector("button.picker-trigger"), null);
    assert.equal(environment.container.querySelector("button button"), null);
    await act(async () => reaction.click());
    assert.deepEqual(reactionClicks, ["👍"]);
    assert.deepEqual(toggleCalls, []);
  } finally {
    await environment.unmount();
  }
});

void test("message examples retain their message and interactive reaction count", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const examples = await import("../../packages/propel/src/emoji-reaction/emoji-reaction.stories.tsx");
  await environment.render(React.createElement(examples.InMessageContext.render));
  assert.match(environment.container.textContent, /Alice Brown.*project launch next week/s);
  const reaction = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("👍")
  );
  assert.ok(reaction);
  assert.match(reaction.textContent, /5/);
  await act(async () => reaction.click());
  assert.match(reaction.textContent, /6/);
  const pickerExamples = await import("../../packages/propel/src/emoji-reaction/emoji-reaction-picker.stories.tsx");
  await environment.render(React.createElement(pickerExamples.InMessageContext.render));
  assert.match(environment.container.textContent, /Alice Brown.*new dashboard/s);
  assert.ok(environment.container.querySelectorAll("button").length > 1);
});
