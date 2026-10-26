import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";

globalThis.commentHighlightTest = { match: true };
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL?.endsWith("/use-comment-highlight.ts") && specifier === "@plane/hooks") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(
          "export const useHashScroll = () => ({ isHashMatch: globalThis.commentHighlightTest.match });"
        )}`,
      };
    }
    return nextResolve(specifier, context);
  },
});

const { useCommentHighlight } = await import("../../apps/web/core/components/comments/card/use-comment-highlight.ts");

function HighlightProbe() {
  return React.createElement("div", {
    "data-testid": "highlight",
    className: useCommentHighlight("comment-1", "/issues/"),
  });
}

await test("comment hash highlight expires and returns on a later hash entry", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  await environment.render(React.createElement(HighlightProbe));
  const readClass = () => environment.container.querySelector('[data-testid="highlight"]').className;
  assert.equal(readClass(), "border-accent-strong");

  await act(async () => t.mock.timers.tick(8000));
  assert.equal(readClass(), "");

  globalThis.commentHighlightTest.match = false;
  await environment.render(React.createElement(HighlightProbe));
  assert.equal(readClass(), "");

  globalThis.commentHighlightTest.match = true;
  await environment.render(React.createElement(HighlightProbe));
  assert.equal(readClass(), "border-accent-strong");
});
