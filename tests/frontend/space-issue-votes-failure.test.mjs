import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";

registerSpaceModuleHooks();

const toastCalls = [];
globalThis.__spaceIssueVoteToastCalls = toastCalls;
const packageStubs = {
  "next/navigation": `export const useRouter = () => ({ push: () => {} }); export const usePathname = () => "/public/issue"; export const useSearchParams = () => new URLSearchParams();`,
  "@/helpers/query-param-generator": `export const queryParamGenerator = () => ({ queryParam: "" });`,
  "@/hooks/store/use-issue-details": `export const useIssueDetails = () => globalThis.__spaceIssueVoteStore;`,
  "@/hooks/store/use-user": `export const useUser = () => ({ data: { id: "actor" } });`,
  "@/hooks/use-is-in-iframe": `export default () => false;`,
  "@plane/propel/toast": `export const TOAST_TYPE = { ERROR: "error" }; export const setToast = value => globalThis.__spaceIssueVoteToastCalls.push(value);`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

const { IssueVotes } = await import("../../apps/space/components/issues/reactions/issue-vote-reactions.tsx");

test("vote buttons recover from a failed request and show an error toast", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  toastCalls.length = 0;

  let rejectVote;
  const store = {
    peekId: "issue",
    details: { issue: { vote_items: null } },
    addIssueVote: () =>
      new Promise((_, reject) => {
        rejectVote = reject;
      }),
    removeIssueVote: async () => {},
  };
  globalThis.__spaceIssueVoteStore = store;

  await environment.render(React.createElement(IssueVotes, { anchor: "anchor" }));
  const buttons = [...environment.container.querySelectorAll("button")];
  assert.equal(buttons.length, 2, "the real tooltip triggers render both vote buttons");
  assert.ok(buttons.every((button) => !button.disabled));

  await React.act(async () => {
    buttons[0].click();
  });
  assert.ok(
    buttons.every((button) => button.disabled),
    "both buttons stay disabled while the request is pending"
  );

  await React.act(async () => {
    rejectVote(new Error("network unavailable"));
    await Promise.resolve();
  });

  const updatedButtons = [...environment.container.querySelectorAll("button")];
  assert.ok(
    updatedButtons.every((button) => !button.disabled),
    "both buttons become available after failure"
  );
  assert.deepEqual(toastCalls, [
    {
      type: "error",
      title: "Error!",
      message: "Vote could not be saved. Please try again.",
    },
  ]);
});
