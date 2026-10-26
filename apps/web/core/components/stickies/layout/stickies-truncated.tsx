/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams } from "@tanstack/react-router";

// plane utils
import { useTranslation } from "@plane/i18n";
import { cn } from "@plane/utils/common";
// hooks
// components
import { ContentOverflowWrapper } from "../../core/content-overflow-HOC";
import { StickiesLayout } from "./stickies-list";

type StickiesTruncatedProps = {
  handleClose?: () => void;
};

export const StickiesTruncated = function StickiesTruncated(props: StickiesTruncatedProps) {
  const { handleClose = () => {} } = props;
  // navigation
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const { t } = useTranslation();

  return (
    <ContentOverflowWrapper
      maxHeight={620}
      containerClassName="pb-2 box-border"
      fallback={null}
      customButton={
        <Link
          to="/$workspaceSlug/stickies/"
          params={{ workspaceSlug: workspaceSlug }}
          className={cn(
            "w-full gap-1 bg-surface-2/20 text-13 font-medium text-accent-primary transition-opacity duration-300"
          )}
          onClick={handleClose}
        >
          {t("show_all")}
        </Link>
      }
    >
      <StickiesLayout workspaceSlug={workspaceSlug?.toString()} />
    </ContentOverflowWrapper>
  );
};
