/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";
import type { IUserLite } from "@plane/types";
import { getFileURL } from "@plane/utils/file";

type Props = {
  member: Pick<IUserLite, "id" | "avatar_url" | "display_name" | "email">;
  workspaceSlug: string;
  fallbackTextClassName: string;
};

export function MemberProfileAvatar({ member, workspaceSlug, fallbackTextClassName }: Props) {
  const { id, avatar_url, display_name, email } = member;
  const avatar = avatar_url && avatar_url.trim() !== "" ? avatar_url : undefined;
  return (
    <Link to="/$workspaceSlug/profile/$userId/" params={{ workspaceSlug, userId: id }}>
      {avatar ? (
        <span className="relative flex size-6 items-center justify-center rounded-full text-on-color capitalize">
          <img
            src={getFileURL(avatar)}
            className="absolute top-0 left-0 h-full w-full rounded-full object-cover"
            alt={display_name || email}
          />
        </span>
      ) : (
        <span
          className={`relative flex size-6 items-center justify-center rounded-full bg-layer-3 text-11 ${fallbackTextClassName} capitalize`}
        >
          {(email ?? display_name ?? "?")[0]}
        </span>
      )}
    </Link>
  );
}
