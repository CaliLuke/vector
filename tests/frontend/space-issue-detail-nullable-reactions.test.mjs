import assert from "node:assert/strict";
import test from "node:test";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";

registerSpaceModuleHooks();

const { IssueDetailStore } = await import("../../apps/space/store/issue-detail.store.ts");

function createStore() {
  const store = new IssueDetailStore({});
  store.rootStore = {
    publishList: {
      publishMap: {
        anchor: { project: "project", workspace_detail: { slug: "workspace" } },
      },
    },
    user: {
      currentActor: { id: "actor", display_name: "Actor" },
      data: { id: "actor" },
    },
  };
  store.details = {
    issue: {
      vote_items: null,
      reaction_items: null,
    },
  };
  return store;
}

test("adding a vote treats a null vote list as empty", async () => {
  const store = createStore();
  store.issueService = {
    async addVote() {},
  };

  await store.addIssueVote("anchor", "issue", { vote: 1 });

  assert.deepEqual(store.details.issue.vote_items, [
    { actor_details: { id: "actor", display_name: "Actor" }, vote: 1 },
  ]);
});

test("removing a vote treats a null vote list as empty", async () => {
  const store = createStore();
  let removed = false;
  store.issueService = {
    async removeVote() {
      removed = true;
    },
  };

  await store.removeIssueVote("anchor", "issue");

  assert.equal(removed, true);
  assert.deepEqual(store.details.issue.vote_items, []);
});

test("adding a reaction treats a null reaction list as empty", async () => {
  const store = createStore();
  store.issueService = {
    async addReaction() {},
    async listReactions() {
      return [];
    },
  };

  await store.addIssueReaction("anchor", "issue", "heart");

  assert.deepEqual(store.details.issue.reaction_items, [
    { actor_details: { id: "actor", display_name: "Actor" }, reaction: "heart" },
  ]);
});

test("removing a reaction treats a null reaction list as empty", async () => {
  const store = createStore();
  let removed = false;
  store.issueService = {
    async removeReaction() {
      removed = true;
    },
    async listReactions() {
      return [];
    },
  };

  await store.removeIssueReaction("anchor", "issue", "heart");

  assert.equal(removed, true);
  assert.deepEqual(store.details.issue.reaction_items, []);
});

test("a failed vote refreshes state and remains observable to the caller", async () => {
  const store = createStore();
  const failure = new Error("request failed");
  let refreshed = false;
  store.issueService = {
    async addVote() {
      throw failure;
    },
    async listVotes() {
      refreshed = true;
      return [];
    },
  };

  await assert.rejects(store.addIssueVote("anchor", "issue", { vote: 1 }), failure);

  assert.equal(refreshed, true);
  assert.deepEqual(store.details.issue.vote_items, []);
});

test("a failed vote removal restores the original votes when refresh also fails", async () => {
  const store = createStore();
  const originalVotes = [
    { actor_details: { id: "actor", display_name: "Actor" }, vote: 1 },
    { actor_details: { id: "other", display_name: "Other" }, vote: -1 },
  ];
  store.details.issue.vote_items = originalVotes;
  const failure = new Error("remove request failed");
  store.issueService = {
    async removeVote() {
      throw failure;
    },
    async listVotes() {
      throw new Error("refresh failed");
    },
  };

  await assert.rejects(store.removeIssueVote("anchor", "issue"), failure);

  assert.deepEqual(store.details.issue.vote_items, originalVotes);
});
