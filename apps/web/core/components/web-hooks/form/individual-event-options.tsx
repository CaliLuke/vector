/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IWebhook } from "@plane/types";
import { Checkbox } from "@plane/ui";

export type WebhookEvents = Pick<IWebhook, "project" | "cycle" | "module" | "issue" | "issue_comment">;

export const INDIVIDUAL_WEBHOOK_OPTIONS: {
  key: keyof WebhookEvents;
  label: string;
  description: string;
}[] = [
  {
    key: "project",
    label: "Projects",
    description: "Project created, updated, or deleted",
  },
  {
    key: "cycle",
    label: "Cycles",
    description: "Cycle created, updated, or deleted",
  },
  {
    key: "issue",
    label: "Work items",
    description: "Work item created, updated, deleted, added to a cycle or module",
  },
  {
    key: "module",
    label: "Modules",
    description: "Module created, updated, or deleted",
  },
  {
    key: "issue_comment",
    label: "Work item comments",
    description: "Comment posted, updated, or deleted",
  },
];

type Props = {
  value: WebhookEvents;
  onChange: (key: keyof WebhookEvents, value: boolean) => void;
  disabled?: boolean;
};

export function WebhookIndividualEventOptions({ value, onChange, disabled }: Props) {
  return (
    <div className="grid grid-cols-1 gap-x-4 gap-y-8 px-6 lg:grid-cols-2">
      {INDIVIDUAL_WEBHOOK_OPTIONS.map((option) => (
        <div key={option.key}>
          <div className="flex items-center gap-2">
            <Checkbox
              id={option.key}
              onChange={() => onChange(option.key, !value[option.key])}
              checked={value[option.key]}
              disabled={disabled}
            />
            <label className="text-13" htmlFor={option.key}>
              {option.label}
            </label>
          </div>
          <p className="mt-0.5 ml-6 text-11 text-tertiary">{option.description}</p>
        </div>
      ))}
    </div>
  );
}
