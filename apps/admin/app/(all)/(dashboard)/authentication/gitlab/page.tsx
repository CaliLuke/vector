/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { OAuthSetupPage } from "@/components/authentication/oauth-setup-page";
import type { Route } from "./+types/page";

export default function GitLabAuthenticationPage() {
  return <OAuthSetupPage provider="gitlab" />;
}
export const meta: Route.MetaFunction = () => [{ title: "GitLab Authentication - God Mode" }];
