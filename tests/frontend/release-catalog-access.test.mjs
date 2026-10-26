import { createProjectQueryDomFixture } from "./helpers/query-dom.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
globalThis.releaseReact = React;
const stubs = {
  "@plane/propel/button": `export const Button=({loading,variant,prependIcon,...props})=>globalThis.releaseReact.createElement("button",props);`,
  "@/hooks/use-current-user": `export const useCurrentUserId=()=>"alice";`,
  "@/hooks/use-permissions": `export const useProjectPermissions=()=>({isPending:false,isError:false,projectRole:20,workspaceRole:15,membershipQuery:{data:{updated_at:"v1"}}});`,
  "@/components/work-item-types/controls": `const R=globalThis.releaseReact;export const inputStyle="";export const Field=({label,children})=>R.createElement("label",null,label,children(label));export const PreviewDialog=({title,children,footer})=>R.createElement("section",null,R.createElement("h2",null,title),children,footer);export const EmptyState=({title,description,children})=>R.createElement("section",null,title,description,children);`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const bootstrap = await createReactDomTestEnvironment();
const { ProjectReleases } = await import("../../apps/web/core/components/releases/project-releases.tsx");
const { useReleaseCatalog } = await import("../../apps/web/core/components/releases/use-release-catalog.ts");
const { projectReleaseOptions } = await import("../../apps/web/core/queries/project-releases.ts");
const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { projectReleasesService: service } = await import("../../apps/web/core/services/project-releases.service.ts");
const { authService } = await import("../../apps/web/core/services/auth.service.ts");
await bootstrap.unmount();
const release = {
  id: "release",
  name: "Private release",
  description: "Internal description",
  target_date: "",
  created_at: "2026-10-25",
};
const catalog = { revision: 1, can_manage: true, project_name: "Project", releases: [release] };
const mutation = {
  id: "release",
  action: "update",
  revision: 1,
  values: { name: "Edited", description: "", target_date: "" },
};
async function fixture(t) {
  const { env, client } = await createProjectQueryDomFixture(t, 20);
  const scope = {
    accountId: "alice",
    generation: 0,
    workspaceSlug: "acme",
    projectId: "p",
    membershipRole: 15,
    membershipRevision: "v1",
    projectRole: 20,
  };
  const f = { env, client, scope, api: [], read: async () => structuredClone(catalog) };
  t.mock.method(service, "catalog", (...args) => f.read(...args));
  function Probe({ index }) {
    const api = useReleaseCatalog(scope);
    React.useLayoutEffect(() => {
      f.api[index] = api;
    });
    return null;
  }
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(
        React.Fragment,
        null,
        React.createElement(ProjectReleases, { workspaceSlug: "acme", projectId: "p" }),
        React.createElement(Probe, { index: 0 }),
        React.createElement(Probe, { index: 1 })
      )
    )
  );
  f.click = (label) =>
    act(async () => [...env.container.querySelectorAll("button")].find((b) => b.textContent === label).click());
  f.refetch = () =>
    act(async () => {
      await f.api[0].query.refetch();
    });
  f.deny = async () => {
    f.read = async () => {
      throw { status: 403 };
    };
    await f.refetch();
  };
  assert.match(env.container.textContent, /Private release/);
  return f;
}
await test("transient background refresh failure retains the mounted release draft", async (t) => {
  const f = await fixture(t);
  await f.click("Edit");
  await changeFormControl(f.env.container.querySelector('[id="Name"]'), "Unsaved local name");
  f.read = async () => {
    throw new Error("Offline");
  };
  await f.refetch();
  assert.equal(f.env.container.querySelector('[id="Name"]').value, "Unsaved local name");
  assert.match(f.env.container.textContent, /Offline/);
});
await test("definitive denial hides all observers and pending or failed retries expose no cached release", async (t) => {
  const f = await fixture(t);
  await f.deny();
  assert.equal(f.api[0].denied, true);
  assert.equal(f.api[1].denied, true);
  assert.doesNotMatch(f.env.container.textContent, /Private release/);
  const request = Promise.withResolvers();
  f.read = () => request.promise;
  let retry;
  await act(async () => {
    retry = f.api[0].retry();
  });
  assert.doesNotMatch(f.env.container.textContent, /Private release/);
  await act(async () => {
    request.reject(new Error("Offline"));
    await retry;
  });
  assert.doesNotMatch(f.env.container.textContent, /Private release/);
  f.read = async () => structuredClone(catalog);
  await act(async () => f.api[0].retry());
  assert.match(f.env.container.textContent, /Private release/);
});
await test("late mutation success cannot resurrect catalog after denial and retry", async (t) => {
  const f = await fixture(t),
    response = Promise.withResolvers();
  t.mock.method(service, "mutate", () => response.promise);
  let save;
  await act(async () => {
    save = f.api[0].save(mutation);
  });
  await f.deny();
  f.read = async () => ({ ...catalog, revision: 9, releases: [] });
  await act(async () => f.api[0].retry());
  await act(async () => {
    response.resolve({ ...catalog, revision: 2 });
    await save;
  });
  assert.equal(f.client.getQueryData(projectReleaseOptions(f.scope).queryKey).revision, 9);
  assert.doesNotMatch(f.env.container.textContent, /Private release/);
});
await test("catalog denial while waiting for CSRF prevents mutation dispatch", async (t) => {
  const f = await fixture(t),
    csrf = Promise.withResolvers();
  let sent = 0;
  t.mock.method(authService, "requestCSRFToken", () => csrf.promise);
  t.mock.method(service, "post", async () => {
    sent++;
    return { data: catalog };
  });
  let result;
  await act(async () => {
    result = f.api[0].save(mutation).then(
      () => null,
      (error) => error
    );
  });
  await f.deny();
  let failure;
  await act(async () => {
    csrf.resolve({ csrf_token: "test" });
    failure = await result;
  });
  assert.match(failure.message, /access changed/);
  assert.equal(sent, 0);
});
await test("an earlier account session cannot publish a late release save", async (t) => {
  const f = await fixture(t),
    response = Promise.withResolvers();
  t.mock.method(service, "mutate", () => response.promise);
  let save;
  await act(async () => {
    save = f.api[0].save(mutation);
  });
  await act(async () => {
    advanceIdentitySession(f.client);
    response.resolve({ ...catalog, revision: 8 });
    await save;
  });
  assert.notEqual(f.client.getQueryData(projectReleaseOptions(f.scope).queryKey)?.revision, 8);
});
