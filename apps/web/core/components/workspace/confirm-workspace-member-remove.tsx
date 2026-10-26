/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
// ui
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";

export type Props = {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: () => Promise<void>;
  userDetails: {
    id: string;
    display_name: string;
  };
};

function getErrorMessage(error: unknown, fallback: string) {
  if (error && typeof error === "object" && "error" in error && typeof error.error === "string") return error.error;
  if (error instanceof Error && error.message) return error.message;
  return error ? fallback : null;
}

function getActionLabel(
  isSelf: boolean,
  isRemoving: boolean,
  labels: { leave: string; leaving: string; remove: string; removing: string }
) {
  if (isSelf) return isRemoving ? labels.leaving : labels.leave;
  return isRemoving ? labels.removing : labels.remove;
}

export function ConfirmWorkspaceMemberRemove(props: Props) {
  const { isOpen, onClose, onSubmit, userDetails } = props;
  const removal = useMutation({ mutationFn: onSubmit, retry: false });
  const isRemoving = removal.isPending;
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { t } = useTranslation();
  const isSelf = currentUser?.id === userDetails.id;

  const handleClose = () => {
    if (isRemoving) return;
    onClose();
  };

  const handleDeletion = () => {
    if (isRemoving) return;
    void removal.mutateAsync().then(onClose, () => undefined);
  };
  const errorMessage = getErrorMessage(removal.error, t("something_went_wrong_please_try_again"));

  return (
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <div className="px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
        <div className="sm:flex sm:items-start">
          <div className="mx-auto flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-danger-subtle sm:mx-0 sm:h-10 sm:w-10">
            <AlertTriangle className="h-6 w-6 text-danger-primary" aria-hidden="true" />
          </div>
          <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
            <h3 className="text-h5-medium leading-6 text-primary">
              {isSelf ? "Leave workspace?" : `Remove ${userDetails.display_name}?`}
            </h3>
            <div className="mt-2">
              {isSelf ? (
                <p className="text-body-xs-regular text-secondary">
                  {t("workspace_settings.settings.members.leave_confirmation")}
                </p>
              ) : (
                <p className="text-body-xs-regular text-secondary">
                  {/* TODO: Add translation here */}
                  Are you sure you want to remove member- <span className="font-bold">{userDetails?.display_name}</span>
                  ? They will no longer have access to this workspace. This action cannot be undone.
                </p>
              )}
            </div>
            {errorMessage && (
              <p role="alert" className="mt-3 text-body-xs-regular text-danger-primary">
                {errorMessage}
              </p>
            )}
          </div>
        </div>
      </div>
      <div className="flex justify-end gap-2 p-4 sm:px-6">
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isRemoving}>
          {t("cancel")}
        </Button>
        <Button variant="error-fill" size="lg" onClick={handleDeletion} loading={isRemoving} disabled={isRemoving}>
          {getActionLabel(isSelf, isRemoving, {
            leave: t("leave"),
            leaving: t("leaving"),
            remove: t("remove"),
            removing: t("removing"),
          })}
        </Button>
      </div>
    </ModalCore>
  );
}
