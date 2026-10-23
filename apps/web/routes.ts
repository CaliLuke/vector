/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { rootRoute, route, index, layout } from "@tanstack/virtual-file-routes";

export const routes = rootRoute("root.tsx", [
  layout("_home", "(home)/layout.tsx", [index("(home)/page.tsx")]),
  route("sign-up", "(all)/sign-up/layout.tsx", [index("(all)/sign-up/page.tsx")]),
  route("accounts/forgot-password", "(all)/accounts/forgot-password/layout.tsx", [
    index("(all)/accounts/forgot-password/page.tsx"),
  ]),
  route("accounts/reset-password", "(all)/accounts/reset-password/layout.tsx", [
    index("(all)/accounts/reset-password/page.tsx"),
  ]),
  route("accounts/set-password", "(all)/accounts/set-password/layout.tsx", [
    index("(all)/accounts/set-password/page.tsx"),
  ]),
  route("create-workspace", "(all)/create-workspace/layout.tsx", [index("(all)/create-workspace/page.tsx")]),
  route("onboarding", "(all)/onboarding/layout.tsx", [index("(all)/onboarding/page.tsx")]),
  route("invitations", "(all)/invitations/layout.tsx", [index("(all)/invitations/page.tsx")]),
  route("workspace-invitations", "(all)/workspace-invitations/layout.tsx", [
    index("(all)/workspace-invitations/page.tsx"),
  ]),
  layout("_authenticated", "(all)/layout.tsx", [
    route("$workspaceSlug", "(all)/[workspaceSlug]/layout.tsx", [
      layout("_projects", "(all)/[workspaceSlug]/(projects)/layout.tsx", [
        index("(all)/[workspaceSlug]/(projects)/page.tsx"),
        route("active-cycles", "(all)/[workspaceSlug]/(projects)/active-cycles/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/active-cycles/page.tsx"),
        ]),
        route("analytics/$tabId", "(all)/[workspaceSlug]/(projects)/analytics/[tabId]/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/analytics/[tabId]/page.tsx"),
        ]),
        route("browse/$workItem", "(all)/[workspaceSlug]/(projects)/browse/[workItem]/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx"),
        ]),
        route("drafts", "(all)/[workspaceSlug]/(projects)/drafts/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/drafts/page.tsx"),
        ]),
        route("notifications", "(all)/[workspaceSlug]/(projects)/notifications/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/notifications/page.tsx"),
        ]),
        route("profile/$userId", "(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/profile/[userId]/page.tsx"),
          route("$profileViewId", "(all)/[workspaceSlug]/(projects)/profile/[userId]/[profileViewId]/page.tsx"),
          route("activity", "(all)/[workspaceSlug]/(projects)/profile/[userId]/activity/page.tsx"),
        ]),
        route("stickies", "(all)/[workspaceSlug]/(projects)/stickies/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/stickies/page.tsx"),
        ]),
        route("workspace-views", "(all)/[workspaceSlug]/(projects)/workspace-views/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/workspace-views/page.tsx"),
          route("$globalViewId", "(all)/[workspaceSlug]/(projects)/workspace-views/[globalViewId]/page.tsx"),
        ]),
        route("projects/archives", "(all)/[workspaceSlug]/(projects)/projects/(detail)/archives/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/projects/(detail)/archives/page.tsx"),
        ]),
        route("projects", "(all)/[workspaceSlug]/(projects)/projects/(list)/layout.tsx", [
          index("(all)/[workspaceSlug]/(projects)/projects/(list)/page.tsx"),
        ]),
        route("projects/$projectId", "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx", [
          route("issues", "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/issues/(list)/layout.tsx", [
            index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/issues/(list)/page.tsx"),
          ]),
          route(
            "issues/$issueId",
            "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/issues/(detail)/[issueId]/page.tsx"
          ),
          route(
            "cycles/$cycleId",
            "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/layout.tsx",
            [index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/[cycleId]/page.tsx")]
          ),
          route("cycles", "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(list)/layout.tsx", [
            index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(list)/page.tsx"),
          ]),
          route(
            "modules/$moduleId",
            "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/layout.tsx",
            [
              index(
                "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/[moduleId]/page.tsx"
              ),
            ]
          ),
          route("modules", "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(list)/layout.tsx", [
            index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(list)/page.tsx"),
          ]),
          route(
            "views/$viewId",
            "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(detail)/layout.tsx",
            [index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(detail)/[viewId]/page.tsx")]
          ),
          route("views", "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(list)/layout.tsx", [
            index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(list)/page.tsx"),
          ]),
          route(
            "pages/$pageId",
            "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/pages/(detail)/layout.tsx",
            [index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/pages/(detail)/[pageId]/page.tsx")]
          ),
          route("pages", "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/pages/(list)/layout.tsx", [
            index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/pages/(list)/page.tsx"),
          ]),
          route("intake", "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/layout.tsx", [
            index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx"),
          ]),
        ]),
        route(
          "projects/$projectId/archives/issues",
          "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/issues/(list)/layout.tsx",
          [index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/issues/(list)/page.tsx")]
        ),
        route(
          "projects/$projectId/archives/issues/$archivedIssueId",
          "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/issues/(detail)/layout.tsx",
          [
            index(
              "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/issues/(detail)/[archivedIssueId]/page.tsx"
            ),
          ]
        ),
        route(
          "projects/$projectId/archives/cycles",
          "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/cycles/layout.tsx",
          [index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/cycles/page.tsx")]
        ),
        route(
          "projects/$projectId/archives/modules",
          "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/modules/layout.tsx",
          [index("(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/modules/page.tsx")]
        ),
      ]),
      route("settings", "(all)/[workspaceSlug]/(settings)/layout.tsx", [
        layout("_workspace", "(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx", [
          index("(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx"),
          route("members", "(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx"),
          route("billing", "(all)/[workspaceSlug]/(settings)/settings/(workspace)/billing/page.tsx"),
          route("exports", "(all)/[workspaceSlug]/(settings)/settings/(workspace)/exports/page.tsx"),
          route("webhooks", "(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx"),
          route(
            "webhooks/$webhookId",
            "(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/[webhookId]/page.tsx"
          ),
        ]),
        route("projects", "(all)/[workspaceSlug]/(settings)/settings/projects/layout.tsx", [
          index("(all)/[workspaceSlug]/(settings)/settings/projects/page.tsx"),
          route("$projectId", "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx", [
            index("(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx"),
            route("members", "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx"),
            route(
              "features/cycles",
              "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx"
            ),
            route(
              "features/modules",
              "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx"
            ),
            route(
              "features/views",
              "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx"
            ),
            route(
              "features/pages",
              "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/pages/page.tsx"
            ),
            route(
              "features/intake",
              "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx"
            ),
            route("states", "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx"),
            route("labels", "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx"),
            route("estimates", "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/estimates/page.tsx"),
            route(
              "automations",
              "(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/layout.tsx",
              [index("(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx")]
            ),
          ]),
        ]),
      ]),
    ]),
    route("settings/profile/$profileTabId", "(all)/settings/profile/layout.tsx", [
      index("(all)/settings/profile/[profileTabId]/page.tsx"),
    ]),
  ]),
  route("$workspaceSlug/projects/$projectId/settings/$", "routes/redirects/core/project-settings.tsx"),
  route("$workspaceSlug/analytics", "routes/redirects/core/analytics.tsx"),
  route("$workspaceSlug/settings/api-tokens", "routes/redirects/core/api-tokens.tsx"),
  route("$workspaceSlug/projects/$projectId/inbox", "routes/redirects/core/inbox.tsx"),
  route("accounts/sign-up", "routes/redirects/core/accounts-signup.tsx"),
  route("sign-in", "routes/redirects/core/sign-in.tsx"),
  route("signin", "routes/redirects/core/signin.tsx"),
  route("login", "routes/redirects/core/login.tsx"),
  route("register", "routes/redirects/core/register.tsx"),
  route("profile/$", "routes/redirects/core/profile-settings.tsx"),
  route("$workspaceSlug/settings/account/$", "routes/redirects/core/workspace-account-settings.tsx"),
]);
