/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { PROJECT_ERROR_MESSAGES } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";

export function isIssuePermissionScopeReady({
  hasAccount,
  identityVerified,
  currentUserLoaded,
  permissionsPending,
  permissionsError,
}: {
  hasAccount: boolean;
  identityVerified: boolean;
  currentUserLoaded: boolean;
  permissionsPending: boolean;
  permissionsError: boolean;
}) {
  return hasAccount && identityVerified && currentUserLoaded && !permissionsPending && !permissionsError;
}

export function isIssueCreator(accountId: string | undefined, creatorId: string | undefined) {
  return Boolean(accountId && creatorId && accountId === creatorId);
}

export function canDeleteIssue({
  canStart,
  hasProjectRole,
  isCreator,
  isAdmin,
}: {
  canStart: boolean;
  hasProjectRole: boolean;
  isCreator: boolean;
  isAdmin: boolean;
}) {
  return canStart && hasProjectRole && (isCreator || isAdmin);
}

export async function submitIssueArchiveConfirmation({
  canArchive,
  isPending,
  setPending,
  isCurrent,
  onSubmit,
  onFailure,
  onSuccess,
}: {
  canArchive: boolean;
  isPending: () => boolean;
  setPending: (pending: boolean) => void;
  isCurrent: () => boolean;
  onSubmit?: () => Promise<void>;
  onFailure: () => void;
  onSuccess: () => void;
}) {
  if (!canArchive || isPending() || !onSubmit || !isCurrent()) return;
  setPending(true);
  try {
    await onSubmit();
  } catch {
    if (!isCurrent()) return;
    setPending(false);
    onFailure();
    return;
  }
  if (!isCurrent()) return;
  setPending(false);
  onSuccess();
}

export async function submitIssueDeleteConfirmation({
  authorized,
  canStart,
  isCurrent,
  onSubmit,
  onClose,
  setIsDeleting,
  entity,
  successToast,
  t,
}: {
  authorized: boolean;
  canStart: boolean;
  isCurrent: () => boolean;
  onSubmit?: () => Promise<void>;
  onClose: () => void;
  setIsDeleting: (value: boolean) => void;
  entity: string;
  successToast: { title: string; message: string };
  t: (key: string) => string;
}) {
  if (!canStart || !onSubmit || !isCurrent()) return;
  setIsDeleting(true);
  if (!authorized) {
    if (!isCurrent()) return;
    const permissionError = PROJECT_ERROR_MESSAGES.permissionError;
    setToast({
      title: t(permissionError.i18n_title),
      type: TOAST_TYPE.ERROR,
      message: permissionError.i18n_message && t(permissionError.i18n_message),
    });
    setIsDeleting(false);
    onClose();
    return;
  }
  try {
    await onSubmit();
  } catch (error) {
    if (!isCurrent()) return;
    const errorCode = error && typeof error === "object" && "error" in error ? error.error : undefined;
    const currentError =
      errorCode === `Only admin or creator can delete the ${entity}`
        ? PROJECT_ERROR_MESSAGES.permissionError
        : PROJECT_ERROR_MESSAGES.issueDeleteError;
    setToast({
      title: t(currentError.i18n_title),
      type: TOAST_TYPE.ERROR,
      message: currentError.i18n_message && t(currentError.i18n_message),
    });
    setIsDeleting(false);
    return;
  }
  if (!isCurrent()) return;
  setIsDeleting(false);
  setToast({ type: TOAST_TYPE.SUCCESS, ...successToast });
  onClose();
}
