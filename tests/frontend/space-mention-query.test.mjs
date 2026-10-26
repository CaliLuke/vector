import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";

const state = {
  fetchCurrentUser: async () => {
    throw new Error("unauthenticated");
  },
};
globalThis.vectorMentionQueryTest = state;
registerSpaceModuleHooks();
const require = createRequire(new URL("../../apps/space/package.json", import.meta.url));
const { UserService } = require("@plane/services");
const { currentUserOptions } = await import("../../apps/space/queries/session.ts");
const { useMention } = await import("../../apps/space/hooks/use-mention.tsx");

function Harness({ ref }) {
  const mention = useMention();
  React.useImperativeHandle(ref, () => mention, [mention]);
  return null;
}

async function mountMention(t) {
  t.mock.method(UserService.prototype, "me", () => state.fetchCurrentUser());
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });
  const ref = React.createRef();
  await environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(Harness, { ref })));
  return { client, mention: ref.current };
}

await test("anonymous mention highlighting finishes with no highlights instead of polling indefinitely", async (t) => {
  const { mention } = await mountMention(t);
  assert.deepEqual(await mention.mentionHighlights(), []);
});

await test("mention highlighting reuses the current-user cache loaded by the public-page provider", async (t) => {
  const { client, mention } = await mountMention(t);
  client.setQueryData(currentUserOptions.queryKey, { id: "reader" });
  state.fetchCurrentUser = async () => assert.fail("the cached user should not be requested again");
  assert.deepEqual(await mention.mentionHighlights(), ["reader"]);
});
