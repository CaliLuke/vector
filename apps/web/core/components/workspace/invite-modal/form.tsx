/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Dialog } from "@headlessui/react";

type TInvitationFormProps = {
  title: string;
  description: React.ReactNode;
  children: React.ReactNode;
  onSubmit: () => Promise<void>;
  actions: React.ReactNode;
  className?: string;
};

export const InvitationForm = function InvitationForm(props: TInvitationFormProps) {
  const { title, description, children, actions, onSubmit, className } = props;

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void onSubmit();
      }}
      className={className}
    >
      <div className="space-y-4">
        <Dialog.Title as="h3" className="text-body-md-medium leading-6 text-primary">
          {title}
        </Dialog.Title>
        <div className="text-body-xs-regular text-secondary">{description}</div>
        {children}
      </div>
      {actions}
    </form>
  );
};
