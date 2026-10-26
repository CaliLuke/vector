/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";

import { AlertTriangle } from "lucide-react";
// types
import { Button } from "@plane/propel/button";
import type { IUserLite } from "@plane/types";
// ui
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useCurrentUser } from "@/hooks/use-current-user";

type Props = {
  data: Partial<IUserLite>;
  onSubmit: () => Promise<void>;
  isOpen: boolean;
  onClose: () => void;
};

export const ConfirmProjectMemberRemove = function ConfirmProjectMemberRemove(props: Props) {
  const { data, onSubmit, isOpen, onClose } = props;
  // router
  const { projectId } = useParams({ strict: false });
  const removal = useMutation({ mutationFn: onSubmit, retry: false });
  const isDeleteLoading = removal.isPending;
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;

  const handleClose = () => {
    if (isDeleteLoading) return;
    onClose();
  };

  const handleDeletion = () => {
    if (isDeleteLoading) return;
    removal.mutate(undefined, { onSuccess: onClose });
  };

  if (!projectId) return <></>;

  const isCurrentUser = currentUser?.id === data?.id;
  const currentProjectDetails = projectRows?.find((candidate) => candidate.id === projectId);

  return (
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <div className="bg-surface-1 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
        <div className="sm:flex sm:items-start">
          <div className="mx-auto flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-danger-subtle sm:mx-0 sm:h-10 sm:w-10">
            <AlertTriangle className="h-6 w-6 text-danger-primary" aria-hidden="true" />
          </div>
          <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
            <h3 className="text-16 leading-6 font-medium text-primary">
              {isCurrentUser ? "Leave project?" : `Remove ${data?.display_name}?`}
            </h3>
            <div className="mt-2">
              <p className="text-13 text-secondary">
                {isCurrentUser ? (
                  <>
                    Are you sure you want to leave the <span className="font-bold">{currentProjectDetails?.name}</span>{" "}
                    project? You will be able to join the project if invited again or if it{"'"}s public.
                  </>
                ) : (
                  <>
                    Are you sure you want to remove member- <span className="font-bold">{data?.display_name}</span>?
                    They will no longer have access to this project. This action cannot be undone.
                  </>
                )}
              </p>
            </div>
            {removal.isError && (
              <p role="alert" className="mt-3 text-13 text-danger-primary">
                {isCurrentUser ? "Unable to leave the project." : "Unable to remove the member."} Please try again.
              </p>
            )}
          </div>
        </div>
      </div>
      <div className="flex justify-end gap-2 p-4 sm:px-6">
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isDeleteLoading}>
          Cancel
        </Button>
        <Button
          variant="error-fill"
          size="lg"
          onClick={handleDeletion}
          loading={isDeleteLoading}
          disabled={isDeleteLoading}
        >
          {isCurrentUser ? (isDeleteLoading ? "Leaving..." : "Leave") : isDeleteLoading ? "Removing..." : "Remove"}
        </Button>
      </div>
    </ModalCore>
  );
};
