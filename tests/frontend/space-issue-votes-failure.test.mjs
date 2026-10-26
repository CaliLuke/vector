import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";

const require = createRequire(new URL("../../apps/space/package.json", import.meta.url));
const { MemoryRouter } = createRequire(new URL("../../apps/space/package.json", import.meta.url))("react-router");
const { SitesIssueService } = require("@plane/services");

registerSpaceModuleHooks();

const toastCalls = [];
globalThis.__spaceIssueVoteToastCalls = toastCalls;
const packageStubs = {
  "@/helpers/query-param-generator": `export const queryParamGenerator = () => ({ queryParam: "" });`,
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
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { currentUserOptions } = await import("../../apps/space/queries/session.ts");
  client.setQueryData(currentUserOptions.queryKey, { id: "actor" });
  t.after(() => client.clear());
  t.mock.method(
    SitesIssueService.prototype,
    "addVote",
    () =>
      new Promise((_, reject) => {
        rejectVote = reject;
      })
  );
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(
        MemoryRouter,
        null,
        React.createElement(IssueVotes, { anchor: "anchor", issue: { id: "issue", vote_items: null } })
      )
    )
  );
  const buttons = [...environment.container.querySelectorAll("button")];
  assert.equal(buttons.length, 2, "the real tooltip triggers render both vote buttons");
  assert.ok(buttons.every((button) => !button.disabled));

  await React.act(async () => {
    buttons[0].click();
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  assert.ok(
    buttons.every((button) => button.disabled),
    "both buttons stay disabled while the request is pending"
  );

  await React.act(async () => {
    rejectVote(new Error("network unavailable"));
    await new Promise((resolve) => setTimeout(resolve, 20));
  });

  const updatedButtons = [...environment.container.querySelectorAll("button")];
  assert.ok(
    updatedButtons.every((button) => !button.disabled),
    "both buttons become available after failure"
  );
  assert.deepEqual(toastCalls, [
    {
      type: "error",
      title: "Could not save",
      message: "Please try again.",
    },
  ]);
});
