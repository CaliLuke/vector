import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  registerPermissionUiStubs,
  settleQuery,
} from "./helpers/permission-ui-fixture.mjs";
import { EUserProjectRoles, EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";

globalThis.memberColumnReact = React;
const menuStub = `
  const h=globalThis.memberColumnReact.createElement;
  export const CustomMenu=Object.assign(({children,customButton})=>h("div",null,customButton,children),{
    MenuItem:({children,onClick})=>h("button",{type:"button",onClick},children)
  });
  export const PopoverMenu=({render})=>render();
  export const CustomSelect=()=>null;
`;
registerPermissionUiStubs("/member-columns.tsx", { "@plane/ui": menuStub });
registerPermissionUiStubs("/member-header-column.tsx", {
  "@plane/ui": menuStub,
  "@plane/i18n": "export const useTranslation=()=>({t:key=>'translated:'+key});",
});
registerPermissionUiStubs("/useMemberColumns.tsx", {
  "@plane/i18n": "export const useTranslation=()=>({t:key=>'translated:'+key});",
});

const { NameColumn: ProjectNameColumn } =
  await import("../../apps/web/core/components/project/settings/member-columns.tsx");
const { NameColumn: WorkspaceNameColumn } =
  await import("../../apps/web/core/components/workspace/settings/member-columns.tsx");
const { useProjectColumns } = await import("../../apps/web/core/components/projects/settings/useProjectColumns.tsx");
const { useMemberColumns } = await import("../../apps/web/core/components/workspace/settings/useMemberColumns.tsx");

function member(overrides = {}) {
  return {
    id: "bob",
    first_name: "Bob",
    last_name: "Member",
    display_name: "Bob M",
    email: "bob@example.test",
    avatar_url: "https://cdn.example.test/avatar.png",
    joining_date: "2026-01-02T00:00:00Z",
    ...overrides,
  };
}

function row(profile = member(), overrides = {}) {
  return {
    member: profile,
    original_role: EUserProjectRoles.MEMBER,
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    role: EUserWorkspaceRoles.MEMBER,
    is_active: true,
    ...overrides,
  };
}

function setup(t) {
  return createPermissionUiFixture(t, {
    path: "/acme/settings/projects/project-1/members/",
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    projectRoles: { "project-1": EUserProjectRoles.ADMIN },
  });
}

function button(container, label) {
  return [...container.querySelectorAll("button")].find((node) => node.textContent?.trim() === label);
}

async function registerNameColumnContracts(scope, NameColumn) {
  await test(`${scope} member profile link follows the row identity and resolves the current workspace`, async (t) => {
    const fixture = await setup(t);
    const removed = [];
    await fixture.render(
      React.createElement(NameColumn, {
        rowData: row(),
        workspaceSlug: "acme",
        isAdmin: true,
        currentUser: { id: "alice" },
        setRemoveMemberModal: (value) => removed.push(value.member.id),
      })
    );
    const container = fixture.environment.container;
    assert.equal(container.querySelector("a")?.getAttribute("href"), "/acme/profile/bob/");
    assert.equal(container.querySelector("img")?.getAttribute("src"), "https://cdn.example.test/avatar.png");
    assert.equal(container.querySelector("img")?.getAttribute("alt"), "Bob M");
    await act(async () => button(container, "Remove").click());
    assert.deepEqual(removed, ["bob"]);
    await act(async () => container.querySelector("a").click());
    await settleQuery();
    assert.equal(fixture.router.state.location.pathname, "/acme/profile/bob/");
  });

  await test(`${scope} blank avatar uses the email initial and self actions still select the original row`, async (t) => {
    const fixture = await setup(t);
    const self = row(member({ id: "alice", avatar_url: "   " }));
    const removed = [];
    await fixture.render(
      React.createElement(NameColumn, {
        rowData: self,
        workspaceSlug: "acme",
        isAdmin: false,
        currentUser: { id: "alice" },
        setRemoveMemberModal: (value) => removed.push(value),
      })
    );
    const container = fixture.environment.container;
    assert.equal(Boolean(container.querySelector("img")), false);
    assert.equal(container.querySelector("a")?.textContent, "b");
    await act(async () => button(container, "Leave").click());
    assert.equal(removed[0], self);
    await fixture.render(
      React.createElement(NameColumn, {
        rowData: row(member({ avatar_url: "", email: null, display_name: "Display" })),
        workspaceSlug: "acme",
        isAdmin: false,
        currentUser: { id: "alice" },
        setRemoveMemberModal: () => assert.fail("not authorized"),
      })
    );
    assert.equal(container.querySelector("a")?.textContent, "D");
    assert.equal(Boolean(button(container, "Remove")), false);
    await fixture.render(
      React.createElement(NameColumn, {
        rowData: row(member({ avatar_url: null, email: null, display_name: null })),
        workspaceSlug: "acme",
        isAdmin: false,
        currentUser: undefined,
        setRemoveMemberModal: () => {},
      })
    );
    assert.equal(container.querySelector("a")?.textContent, "?");
  });
}

await registerNameColumnContracts("project", ProjectNameColumn);
await registerNameColumnContracts("workspace", WorkspaceNameColumn);

await test("suspended workspace members have no profile navigation or removal action", async (t) => {
  const fixture = await setup(t);
  await fixture.render(
    React.createElement(WorkspaceNameColumn, {
      rowData: row(member(), { is_active: false }),
      workspaceSlug: "acme",
      isAdmin: true,
      currentUser: { id: "alice" },
      setRemoveMemberModal: () => assert.fail("suspended member action"),
    })
  );
  assert.equal(Boolean(fixture.environment.container.querySelector("a")), false);
  assert.equal(Boolean(button(fixture.environment.container, "Remove")), false);
  assert.match(fixture.environment.container.textContent, /Bob Member/);
});

async function registerTableContract(scope) {
  await test(`${scope} table sorts by full name, clears only sorting, and keeps its member selection`, async (t) => {
    const fixture = await setup(t);
    const subject = row();
    const useColumns =
      scope === "project"
        ? (filters, onUpdate) =>
            useProjectColumns({
              projectId: "project-1",
              workspaceSlug: "acme",
              displayFilters: filters,
              handleDisplayFilterUpdate: onUpdate,
            })
        : (filters, onUpdate) => useMemberColumns({ filters, onFilterUpdate: onUpdate });
    function Table() {
      const [filters, setFilters] = React.useState({ roles: ["15"] });
      const onUpdate = (updates) => setFilters((current) => ({ ...current, ...updates }));
      const table = useColumns(filters, onUpdate);
      return React.createElement(
        React.Fragment,
        null,
        React.createElement("output", { "data-testid": "filters" }, JSON.stringify(filters)),
        React.createElement("output", { "data-testid": "selected" }, table.removeMemberModal?.member.id ?? "none"),
        ...table.columns.map((column) =>
          React.createElement(
            "section",
            { key: column.key, "data-column": column.key },
            React.createElement("h2", null, column.content),
            column.thRender?.(),
            column.tdRender(subject)
          )
        )
      );
    }
    await fixture.render(React.createElement(Table));
    const container = fixture.environment.container;
    const first = () => container.querySelector("section");
    assert.equal(
      first().querySelector("h2").textContent,
      scope === "project" ? "Full name" : "translated:workspace_settings.settings.members.details.full_name"
    );
    const nameMenuButtons = () => [...first().querySelectorAll("button")];
    await act(async () => nameMenuButtons()[0].click());
    assert.deepEqual(JSON.parse(container.querySelector('[data-testid="filters"]').textContent), {
      roles: ["15"],
      order_by: "full_name",
    });
    await act(async () => nameMenuButtons()[1].click());
    assert.deepEqual(JSON.parse(container.querySelector('[data-testid="filters"]').textContent), {
      roles: ["15"],
      order_by: "-full_name",
    });
    await act(async () => button(first(), "translated:common.actions.clear_sorting").click());
    assert.deepEqual(JSON.parse(container.querySelector('[data-testid="filters"]').textContent), { roles: ["15"] });
    await act(async () => button(first(), "Remove").click());
    assert.equal(container.querySelector('[data-testid="selected"]').textContent, "bob");
    if (scope === "workspace") {
      const authentication = container.querySelector('[data-column="Authentication"]');
      assert.equal(authentication.querySelectorAll("button").length, 0, "authentication has no sort menu");
    }
  });
}

await registerTableContract("project");
await registerTableContract("workspace");
