import assert from "node:assert/strict";
import test from "node:test";
import { EUserProjectRoles, EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import { PROJECT_DETAILS } from "../../packages/constants/dist/index.mjs";
import {
  React,
  act,
  createPermissionUiFixture,
  registerPermissionUiStubs,
  settleQuery,
} from "./helpers/permission-ui-fixture.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

globalThis.React = React;
globalThis.projectRosterTestToasts = [];

registerPermissionUiStubs("/member-columns.tsx", {
  "@plane/ui": `
    const h=globalThis.React.createElement;
    export const CustomSelect=Object.assign(({value,onChange,disabled,children,label})=>h("select",{value:String(value??""),disabled,"data-testid":"role-selector",onChange:event=>onChange(Number(event.target.value))},children),{Option:({value,children})=>h("option",{value:String(value)},children)});
    export const CustomMenu=()=>null;
  `,
  "@headlessui/react": `export const Disclosure=({children})=>children({});`,
  "@plane/utils/file": "export const getFileURL=value=>value;",
});
registerPermissionUiStubs("/send-project-invitation-modal.tsx", {
  "@plane/ui": `
    const h=globalThis.React.createElement;
    export const Avatar=()=>null;
    export const CustomSelect=Object.assign(({value,onChange,disabled,children,label})=>h("select",{value:String(value??""),disabled,"data-testid":"role-selector",onChange:event=>onChange(Number(event.target.value))},children),{Option:({value,children})=>h("option",{value:String(value)},children)});
    export const CustomSearchSelect=({value,onChange,disabled,options,label})=>h("select",{value:String(value??""),disabled,"data-testid":"member-selector",onChange:event=>onChange(event.target.value)},[h("option",{key:"empty",value:""},label),...(options??[]).map(option=>h("option",{key:option.value,value:option.value},option.query))]);
    export const ModalCore=({isOpen,children})=>isOpen?h("div",{role:"dialog"},children):null;
    export const EModalPosition={CENTER:"center"};
    export const EModalWidth={XXL:"xxl"};
  `,
  "@plane/propel/button": `
    const h=globalThis.React.createElement;
    export const Button=({children,loading,variant,size,...props})=>h("button",props,loading?"Loading...":children);
  `,
  "@plane/propel/icons": `
    const h=globalThis.React.createElement;
    export const PlusIcon=()=>h("span");
    export const CloseIcon=()=>h("span");
  `,
  "@plane/propel/toast": `
    export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};
    export const setToast=value=>globalThis.projectRosterTestToasts.push(value);
  `,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/utils/file": "export const getFileURL=value=>value;",
});
registerPermissionUiStubs("/member-list.tsx", {
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/button": `export const Button=({children,...props})=>globalThis.React.createElement("button",props,children);`,
  "@plane/propel/icons": `export const SearchIcon=()=>null;`,
  "@/components/ui/loader/settings/members": `export const MembersSettingsLoader=()=>null;`,
  "./dropdowns/filters/member-list": `export const MemberListFiltersDropdown=()=>null;`,
  "./member-list-item": `
    const h=globalThis.React.createElement;
    export const ProjectMemberListItem=({memberDetails,handleDisplayFilterUpdate})=>h("div",null,[
      h("button",{key:"sort",onClick:()=>handleDisplayFilterUpdate({order_by:"joining_date"})},"Sort by joining date"),
      ...memberDetails.map(row=>h("div",{key:row.member.id,"data-testid":"member-row"},row.member.id+":"+row.member.joining_date))
    ]);
  `,
  "./send-project-invitation-modal": `export const SendProjectInvitationModal=()=>null;`,
});

registerPermissionUiStubs("/member-select.tsx", {
  "@plane/ui": `
    const h=globalThis.React.createElement;
    export const Avatar=()=>null;
    export const CustomSearchSelect=({options})=>h("div",null,(options??[]).map(option=>h("button",{key:option.value,"data-testid":"member-option","data-value":option.value},option.query)));
  `,
  "@plane/utils/file": "export const getFileURL=value=>value;",
});

const { ProjectMemberService } = await import("../../apps/web/core/services/project/project-member.service.ts");
const { workspaceMemberRosterOptions, projectMemberRosterOptions } =
  await import("../../apps/web/core/queries/member-rosters.ts");
const { AccountTypeColumn } = await import("../../apps/web/core/components/project/settings/member-columns.tsx");
const { SendProjectInvitationModal } =
  await import("../../apps/web/core/components/project/send-project-invitation-modal.tsx");
const { ProjectMemberList } = await import("../../apps/web/core/components/project/member-list.tsx");
const { MemberSelect } = await import("../../apps/web/core/components/project/member-select.tsx");

const workspaceRow = (id, role = EUserWorkspaceRoles.MEMBER, isActive = true) => ({
  member: {
    id,
    display_name: id === "alice" ? "Alice" : "Bob Member",
    first_name: id === "alice" ? "Alice" : "Bob",
    last_name: id === "alice" ? "Admin" : "Member",
    avatar_url: "",
    is_bot: false,
  },
  role,
  is_active: isActive,
  created_at: "2026-01-01T00:00:00Z",
});

async function seedProjectRoster(fixture, { workspaceRole, projectRole, roster = [workspaceRow("bob")], memberships }) {
  await seedPermissionQueries(fixture.client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole,
    projectRoles: { "project-1": projectRole },
  });
  const revision = "2026-10-24T00:00:00Z";
  fixture.client.setQueryData(PROJECT_DETAILS("acme", "project-1"), { id: "project-1", name: "Project" });
  fixture.client.setQueryData(
    workspaceMemberRosterOptions({
      accountId: "alice",
      workspaceSlug: "acme",
      membershipRole: workspaceRole,
      membershipRevision: revision,
    }).queryKey,
    [workspaceRow("alice", workspaceRole), ...roster]
  );
  fixture.client.setQueryData(
    projectMemberRosterOptions({
      accountId: "alice",
      workspaceSlug: "acme",
      projectId: "project-1",
      membershipRole: workspaceRole,
      membershipRevision: revision,
      projectRole,
    }).queryKey,
    memberships ?? [
      {
        id: "membership-bob",
        member: "bob",
        role: EUserProjectRoles.MEMBER,
        original_role: EUserProjectRoles.MEMBER,
        created_at: "2026-02-01T00:00:00Z",
      },
    ]
  );
}

async function createProjectAdminFixture(t, rosterOptions = {}) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/settings/projects/project-1/members/",
    userId: "alice",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": EUserProjectRoles.ADMIN },
  });
  await seedProjectRoster(fixture, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRole: EUserProjectRoles.ADMIN,
    ...rosterOptions,
  });
  return fixture;
}

function select(testId, value) {
  const node = document.querySelector(`[data-testid="${testId}"]`);
  assert.ok(node, `missing ${testId}`);
  node.value = String(value);
  node.dispatchEvent(new Event("change", { bubbles: true }));
}

function button(label) {
  return [...document.querySelectorAll("button")].find((node) => node.textContent === label);
}

async function renderWithNativeQueryContext(fixture, element) {
  await fixture.render(element);
}

await test("project member role picker caps requested roles below a project-admin actor", async (t) => {
  const fixture = await createProjectAdminFixture(t);
  await renderWithNativeQueryContext(
    fixture,
    React.createElement(AccountTypeColumn, {
      rowData: {
        member: workspaceRow("bob").member,
        workspaceRole: EUserWorkspaceRoles.MEMBER,
        original_role: EUserProjectRoles.MEMBER,
      },
      workspaceSlug: "acme",
      projectId: "project-1",
    })
  );
  const roleOptions = [...document.querySelectorAll('[data-testid="role-selector"] option')].map(
    (option) => option.value
  );
  assert.deepEqual(roleOptions, ["5", "15"]);
});

await test("project invitation retains selections after failure and retries the scoped bulk write", async (t) => {
  const fixture = await createProjectAdminFixture(t, {
    roster: [
      workspaceRow("bob"),
      { ...workspaceRow("service-bot"), member: { ...workspaceRow("service-bot").member, is_bot: true } },
    ],
    memberships: [],
  });
  const attempts = [];
  let calls = 0;
  t.mock.method(ProjectMemberService.prototype, "bulkAddMembersToProject", async (_slug, _projectId, payload) => {
    attempts.push(payload);
    calls++;
    if (calls === 1) throw { error: "Temporary failure" };
    return [
      {
        id: "membership-bob",
        member: "bob",
        role: EUserProjectRoles.MEMBER,
        original_role: EUserProjectRoles.MEMBER,
        created_at: "2026-02-01T00:00:00Z",
      },
    ];
  });
  let closed = 0;
  await renderWithNativeQueryContext(
    fixture,
    React.createElement(SendProjectInvitationModal, {
      isOpen: true,
      workspaceSlug: "acme",
      projectId: "project-1",
      onClose: () => closed++,
    })
  );
  assert.equal(document.querySelector('[data-testid="member-selector"] option[value="service-bot"]'), null);
  await act(async () => select("member-selector", "bob"));
  await settleQuery();
  await act(async () => select("role-selector", EUserProjectRoles.MEMBER));
  await act(async () => button("add_members").click());
  await settleQuery();
  assert.equal(calls, 1);
  assert.equal(closed, 0);
  assert.match(document.querySelector('[role="alert"]')?.textContent ?? "", /Could not add members/);
  assert.equal(document.querySelector('[data-testid="member-selector"]').value, "bob");

  await act(async () => button("add_members").click());
  await settleQuery();
  assert.equal(calls, 2);
  assert.equal(closed, 1);
  assert.deepEqual(attempts[0], attempts[1]);
  assert.deepEqual(attempts[1], { members: [{ member_id: "bob", role: EUserProjectRoles.MEMBER }] });
});

await test("project member sorting keeps the project membership date when native workspace rows lack joining_date", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/settings/projects/project-1/members/",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": EUserProjectRoles.ADMIN },
  });
  await seedProjectRoster(fixture, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRole: EUserProjectRoles.ADMIN,
    roster: [workspaceRow("bob"), workspaceRow("carol")],
    memberships: [
      { id: "membership-bob", member: "bob", role: EUserProjectRoles.MEMBER, created_at: "2026-02-01T00:00:00Z" },
      { id: "membership-carol", member: "carol", role: EUserProjectRoles.MEMBER, created_at: "2026-01-01T00:00:00Z" },
    ],
  });
  await fixture.render(React.createElement(ProjectMemberList, { workspaceSlug: "acme", projectId: "project-1" }));
  await settleQuery();
  await act(async () => button("Sort by joining date").click());
  const rows = [...document.querySelectorAll('[data-testid="member-row"]')].map((row) => row.textContent);
  assert.deepEqual(rows, ["carol:2026-01-01T00:00:00Z", "bob:2026-02-01T00:00:00Z"]);
});

await test("project default member selector exposes roster read failures and retries", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/settings/projects/project-1/members/",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": EUserProjectRoles.ADMIN },
  });
  await seedProjectRoster(fixture, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRole: EUserProjectRoles.ADMIN,
  });
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "project-1",
    membershipRole: EUserWorkspaceRoles.MEMBER,
    membershipRevision: "2026-10-24T00:00:00Z",
    projectRole: EUserProjectRoles.ADMIN,
  };
  fixture.client.removeQueries({ queryKey: projectMemberRosterOptions(scope).queryKey, exact: true });
  let attempts = 0;
  t.mock.method(ProjectMemberService.prototype, "fetchProjectMembers", async () => {
    attempts++;
    if (attempts === 1) throw new Error("Temporary roster read failure");
    return [
      {
        id: "membership-bob",
        member: "bob",
        role: EUserProjectRoles.MEMBER,
        original_role: EUserProjectRoles.MEMBER,
        created_at: "2026-02-01T00:00:00Z",
      },
    ];
  });

  await fixture.render(
    React.createElement(MemberSelect, {
      value: undefined,
      onChange: () => {},
      workspaceSlug: "acme",
      projectId: "project-1",
    })
  );
  await settleQuery();
  assert.match(document.querySelector('[role="alert"]')?.textContent ?? "", /Unable to load project members/);
  await act(async () => button("Retry").click());
  await settleQuery();
  assert.equal(attempts, 2);
  assert.equal(document.querySelector('[role="alert"]'), null);
  assert.equal(document.querySelector('[data-testid="member-option"][data-value="bob"]')?.textContent, "Bob Member");
});
