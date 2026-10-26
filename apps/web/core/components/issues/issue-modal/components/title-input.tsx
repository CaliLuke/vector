/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { issueTitleValidators, type IssueDraftForm } from "../form-state";
import React from "react";

// plane imports
import { ETabIndices } from "@plane/constants";
// types
import { useTranslation } from "@plane/i18n";
// ui
import { Input } from "@plane/ui";
// helpers
import { getTabIndex } from "@plane/utils/tab-indices";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";

type TIssueTitleInputProps = {
  form: IssueDraftForm;
  required?: boolean;
  issueTitleRef: React.MutableRefObject<HTMLInputElement | null>;
};

export const IssueTitleInput = function IssueTitleInput(props: TIssueTitleInputProps) {
  const { form, issueTitleRef, required = true } = props;
  // store hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();

  const { getIndex } = getTabIndex(ETabIndices.ISSUE_FORM, isMobile);

  return (
    <div data-issue-field="name">
      <form.Field
        name="name"
        validators={issueTitleValidators(
          t("title_is_required"),
          t("title_should_be_less_than_255_characters"),
          required
        )}
      >
        {(field) => (
          <>
            <Input
              id="name"
              name="name"
              type="text"
              value={field.state.value}
              onChange={(e) => {
                field.handleChange(e.target.value);
              }}
              ref={issueTitleRef}
              hasError={!field.state.meta.isValid}
              placeholder={t("title")}
              className="w-full text-body-sm-regular"
              tabIndex={getIndex("name")}
            />
            {[...new Set(field.state.meta.errors)].map((message) => (
              <p key={message} role="alert" className="text-11 text-danger-primary">
                {message}
              </p>
            ))}
          </>
        )}
      </form.Field>
    </div>
  );
};
