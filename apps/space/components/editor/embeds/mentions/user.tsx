/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/session";

// helpers
import { cn } from "@plane/utils/common";
// hooks
import { membersOptions } from "@/queries/lookups";

type Props = {
  id: string;
};

export const EditorUserMention = function EditorUserMention(props: Props) {
  const { id } = props;
  const { data: currentUser } = useQuery(currentUserOptions);
  const { data: members = [] } = useQuery(membersOptions(useParams().anchor));
  // derived values
  const userDetails = members.find((item) => item.member === id);

  if (!userDetails) {
    return (
      <div className="not-prose inline rounded-sm bg-layer-1 px-1 py-0.5 text-tertiary no-underline">
        @deactivated user
      </div>
    );
  }

  return (
    <div
      className={cn("not-prose inline rounded-sm bg-accent-primary/20 px-1 py-0.5 text-accent-primary no-underline", {
        "bg-yellow-500/20 text-yellow-500": id === currentUser?.id,
      })}
    >
      @{userDetails?.member__display_name}
    </div>
  );
};
