/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
import { useParams, useRouter } from "@tanstack/react-router";

import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IProjectView } from "@plane/types";
import { AlertModalCore } from "@plane/ui";
import { useProjectViewMutations } from "@/hooks/use-project-views";

type Props = {
  data: IProjectView;
  isOpen: boolean;
  onClose: () => void;
};

export function DeleteProjectViewModal(props: Props) {
  if (!props.isOpen) return null;
  return <DeleteProjectViewModalContent key={props.data.id} {...props} />;
}

function DeleteProjectViewModalContent(props: Props) {
  const { data, onClose } = props;
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const router = useRouter();
  const { remove } = useProjectViewMutations(workspaceSlug?.toString(), projectId?.toString());
  const { t } = useTranslation();
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const handleClose = () => {
    onClose();
  };

  const handleDeleteView = async () => {
    if (!workspaceSlug || !projectId || remove.isPending) return;
    try {
      await remove.mutateAsync(data.id);
      if (!mounted.current) return;
      handleClose();
      void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/views` });
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "View deleted successfully.",
      });
    } catch (_error) {
      if (mounted.current) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "View could not be deleted. Please try again.",
        });
      }
    }
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDeleteView}
      isSubmitting={remove.isPending}
      isOpen
      title={t("project_views.delete_view.title")}
      content={<>{t("project_views.delete_view.content")}</>}
    />
  );
}
