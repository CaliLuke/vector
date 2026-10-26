import type { TIssue } from "@plane/types";

export type InboxIssueConfirmationProps = {
  data: Partial<TIssue>;
  projectIdentifier?: string;
  isOpen: boolean;
  onClose: () => void;
  onSubmit: () => Promise<void>;
  onSuccess?: () => void;
  isCurrent: () => boolean;
};
