import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import {
  setupAdminSettingsFormTest,
  adminRequire,
  changeInput,
  formButton,
  formToasts,
  submitAndWait,
} from "./helpers/admin-forms.mjs";
const workspaceFormSource = "apps/admin/app/(all)/(dashboard)/workspace/create/form.tsx";
const workspaceFormObserver = setupAdminSettingsFormTest([workspaceFormSource]);
const { InstanceWorkspaceService } = adminRequire("@plane/services");

await test("workspace form preserves an edited slug and retries unavailable URL checks before creation", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { WorkspaceCreateForm } = await import("../../apps/admin/app/(all)/(dashboard)/workspace/create/form.tsx");
  const { unstable_HistoryRouter: HistoryRouter, UNSAFE_createMemoryHistory: createMemoryHistory } =
    adminRequire("react-router");
  const history = createMemoryHistory({ initialEntries: ["/workspace/create"] });
  let failNavigation = false;
  const push = history.push.bind(history);
  history.push = (...args) => {
    if (failNavigation) {
      failNavigation = false;
      throw new Error("navigation unavailable");
    }
    return push(...args);
  };
  const client = new QueryClient();
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });
  let checkResult = "offline";
  let slugChecks = 0;
  let failCreate = true;
  const payloads = [];
  t.mock.method(InstanceWorkspaceService.prototype, "slugCheck", async () => {
    slugChecks++;
    if (checkResult === "offline") throw new Error("offline");
    return { status: checkResult === "available" };
  });
  t.mock.method(InstanceWorkspaceService.prototype, "create", async (payload) => {
    payloads.push(payload);
    if (failCreate) throw new Error("offline");
    return { ...payload, id: "created" };
  });
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(HistoryRouter, { history }, React.createElement(WorkspaceCreateForm))
    )
  );
  const name = environment.container.querySelector('[name="name"]');
  const slug = environment.container.querySelector('[name="slug"]');
  await changeInput(name, "First workspace");
  assert.equal(slug.value, "first-workspace");
  await changeInput(slug, "custom-url");
  await changeInput(name, "Renamed workspace");
  assert.equal(slug.value, "custom-url");
  const size = environment.container.querySelector("select");
  await act(async () => {
    size.value = size.options[1].value;
    size.dispatchEvent(new Event("change", { bubbles: true }));
  });
  const formApi = workspaceFormObserver.getForms(workspaceFormSource).at(-1);
  const submit = () => submitAndWait(formButton(environment.container, "Create workspace"), formApi);
  await submit();
  assert.equal(payloads.length, 0);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Could not verify/);
  assert.equal(formButton(environment.container, "Create workspace").disabled, false);
  checkResult = "taken";
  await submit();
  assert.equal(payloads.length, 0);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /taken/i);
  checkResult = "available";
  await submit();
  assert.equal(payloads.length, 1);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(formToasts.at(-1).type, "error");
  failCreate = false;
  failNavigation = true;
  await submit();
  assert.equal(payloads.length, 2, "create is retried after its first request fails");
  assert.equal(slugChecks, 4);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(formToasts.at(-1).title, "Workspace created");
  assert.match(formToasts.at(-1).message, /could not open it/);
  assert.equal(formButton(environment.container, "Create workspace").disabled, false);
  await submit();
  assert.equal(payloads.length, 2, "retry after a durable create does not create a duplicate");
  assert.equal(slugChecks, 4, "retry after a durable create does not recheck the slug");
  assert.equal(history.location.pathname, "/workspace");
  assert.deepEqual(payloads, [
    { name: "Renamed workspace", slug: "custom-url", organization_size: size.value },
    { name: "Renamed workspace", slug: "custom-url", organization_size: size.value },
  ]);
  assert.equal(formApi.state.isSubmitSuccessful, true);
});
