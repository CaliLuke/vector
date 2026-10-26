import assert from "node:assert/strict";
import test from "node:test";
import { EUserPermissions } from "../../packages/types/dist/index.mjs";
import { registerWebModuleHooks } from "../../tests/frontend/helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { projectRosterMemberIds, workspaceRosterMemberIds } = await import("../../apps/web/helpers/member-rosters.ts");

const workspaceMember = (id, displayName, { bot = false, active = true } = {}) => ({
  id: `workspace-${id}`,
  role: 15,
  is_active: active,
  member: { id, display_name: displayName, is_bot: bot },
});

await test("workspace roster options filter bots and keep the current user first", () => {
  const members = [
    workspaceMember("zara", "Zara"),
    workspaceMember("bot", "Bot", { bot: true }),
    workspaceMember("alice", "alice"),
    workspaceMember("self", "Current user", { active: false }),
  ];

  assert.deepEqual(workspaceRosterMemberIds(members, "self"), ["self", "alice", "zara"]);
});

await test("project roster options apply guest exclusion and sort unavailable summaries last", () => {
  const memberships = [
    { id: "m-z", member: "zara", role: EUserPermissions.MEMBER, original_role: 15, created_at: "1" },
    { id: "m-g", member: "guest", role: EUserPermissions.GUEST, original_role: 5, created_at: "2" },
    { id: "m-a", member: "alice", role: EUserPermissions.MEMBER, original_role: 15, created_at: "3" },
    { id: "m-x", member: "missing-summary", role: EUserPermissions.MEMBER, original_role: 15, created_at: "4" },
  ];
  const membersById = new Map([
    ["zara", workspaceMember("zara", "Zara")],
    ["alice", workspaceMember("alice", "Alice")],
  ]);

  assert.deepEqual(projectRosterMemberIds(memberships, membersById, "zara", false), [
    "zara",
    "alice",
    "missing-summary",
  ]);
  assert.deepEqual(projectRosterMemberIds(memberships, membersById, "zara", true), [
    "zara",
    "alice",
    "guest",
    "missing-summary",
  ]);
});
