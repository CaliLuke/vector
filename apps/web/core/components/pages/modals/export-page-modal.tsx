/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import type { PageProps } from "@react-pdf/renderer";
import { pdf } from "@react-pdf/renderer";
import { useForm, useStore } from "@tanstack/react-form";

// plane editor
import type { EditorRefApi } from "@plane/editor";
// plane ui
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { CustomSelect, EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// components
import { PDFDocument } from "@/components/editor/pdf";
// hooks
import { useParseEditorContent } from "@/hooks/use-parse-editor-content";

type Props = {
  editorRef: EditorRefApi | null;
  isOpen: boolean;
  onClose: () => void;
  pageTitle: string;
};

type TExportFormats = "pdf" | "markdown";
type TPageFormats = Exclude<PageProps["size"], undefined>;
type TContentVariety = "everything" | "no-assets";

type TFormValues = {
  export_format: TExportFormats;
  page_format: TPageFormats;
  content_variety: TContentVariety;
};

const EXPORT_FORMATS: {
  key: TExportFormats;
  label: string;
}[] = [
  {
    key: "pdf",
    label: "PDF",
  },
  {
    key: "markdown",
    label: "Markdown",
  },
];

const PAGE_FORMATS: {
  key: TPageFormats;
  label: string;
}[] = [
  {
    key: "A4",
    label: "A4",
  },
  {
    key: "A3",
    label: "A3",
  },
  {
    key: "A2",
    label: "A2",
  },
  {
    key: "LETTER",
    label: "Letter",
  },
  {
    key: "LEGAL",
    label: "Legal",
  },
  {
    key: "TABLOID",
    label: "Tabloid",
  },
];

const CONTENT_VARIETY: {
  key: TContentVariety;
  label: string;
}[] = [
  {
    key: "everything",
    label: "Everything",
  },
  {
    key: "no-assets",
    label: "No images",
  },
];

const defaultValues: TFormValues = {
  export_format: "pdf",
  page_format: "A4",
  content_variety: "everything",
};

export function ExportPageModal(props: Props) {
  return props.isOpen ? <ExportPageDialog {...props} /> : null;
}

function ExportPageDialog({ editorRef, onClose, pageTitle }: Props) {
  // params
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // parse editor content
  const {
    memberRosterQuery,
    memberRosterIsLoading,
    memberRosterIsError,
    retryMemberRoster,
    replaceCustomComponentsFromHTMLContent,
    replaceCustomComponentsFromMarkdownContent,
  } = useParseEditorContent({
    projectId,
    workspaceSlug: workspaceSlug ?? "",
  });
  const isMemberRosterReady = memberRosterQuery.isSuccess && !memberRosterIsError;
  // form info
  const form = useForm({
    defaultValues,
    validators: {
      onSubmit: () => (isMemberRosterReady ? undefined : "Load workspace members before exporting."),
    },
    onSubmit: async () => {
      await handleExport();
    },
  });
  const isExporting = useStore(form.store, (state) => state.isSubmitting);
  const values = useStore(form.store, (state) => state.values);
  // derived values
  const selectedExportFormat = values.export_format;
  const selectedPageFormat = values.page_format;
  const selectedContentVariety = values.content_variety;
  const isPDFSelected = selectedExportFormat === "pdf";
  const fileName = pageTitle
    ?.toLowerCase()
    ?.replace(/[^a-z0-9-_]/g, "-")
    .replace(/-+/g, "-");
  // handle modal close
  const handleClose = () => {
    if (!isExporting) onClose();
  };

  const initiateDownload = (blob: Blob, filename: string) => {
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    setTimeout(() => {
      URL.revokeObjectURL(url);
    }, 1000);
  };

  // handle export as a PDF
  const handleExportAsPDF = async () => {
    try {
      const pageContent = `<h1 class="page-title">${pageTitle}</h1>${editorRef?.getDocument().html ?? "<p></p>"}`;
      const parsedPageContent = await replaceCustomComponentsFromHTMLContent({
        htmlContent: pageContent,
        noAssets: selectedContentVariety === "no-assets",
      });

      const blob = await pdf(<PDFDocument content={parsedPageContent} pageFormat={selectedPageFormat} />).toBlob();
      initiateDownload(blob, `${fileName}-${selectedPageFormat.toString().toLowerCase()}.pdf`);
    } catch (error) {
      throw new Error("Error in exporting as a PDF.", { cause: error });
    }
  };
  // handle export as markdown
  const handleExportAsMarkdown = async () => {
    try {
      const markdownContent = editorRef?.getMarkDown() ?? "";
      const parsedMarkdownContent = replaceCustomComponentsFromMarkdownContent({
        markdownContent,
        noAssets: selectedContentVariety === "no-assets",
      });

      const blob = new Blob([parsedMarkdownContent], { type: "text/markdown" });
      initiateDownload(blob, `${fileName}.md`);
    } catch (error) {
      throw new Error("Error in exporting as markdown.", { cause: error });
    }
  };
  // handle export
  const handleExport = async () => {
    if (!isMemberRosterReady) throw new Error("Load workspace members before exporting.");
    try {
      if (selectedExportFormat === "pdf") {
        await handleExportAsPDF();
      }
      if (selectedExportFormat === "markdown") {
        await handleExportAsMarkdown();
      }
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Page exported successfully.",
      });
      onClose();
    } catch (error) {
      console.error("Error in exporting page:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Page could not be exported. Please try again later.",
      });
      throw error;
    }
  };

  return (
    <ModalCore isOpen handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.SM}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit().catch(() => {
            // The export failure is already reported with a toast.
          });
        }}
      >
        <div className="space-y-5 p-5">
          <h3 className="text-18 font-medium text-secondary">Export page</h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-2">
              <h6 className="flex-shrink-0 text-13 text-secondary">Export format</h6>
              <form.Field name="export_format">
                {(field) => (
                  <CustomSelect
                    label={EXPORT_FORMATS.find((format) => format.key === field.state.value)?.label}
                    disabled={isExporting}
                    buttonClassName="border-none"
                    value={field.state.value}
                    onChange={(val: TExportFormats) => field.handleChange(val)}
                    className="flex-shrink-0"
                    placement="bottom-end"
                  >
                    {EXPORT_FORMATS.map((format) => (
                      <CustomSelect.Option key={format.key} value={format.key}>
                        {format.label}
                      </CustomSelect.Option>
                    ))}
                  </CustomSelect>
                )}
              </form.Field>
            </div>
            <div className="flex items-center justify-between gap-2">
              <h6 className="flex-shrink-0 text-13 text-secondary">Include content</h6>
              <form.Field name="content_variety">
                {(field) => (
                  <CustomSelect
                    label={CONTENT_VARIETY.find((variety) => variety.key === field.state.value)?.label}
                    disabled={isExporting}
                    buttonClassName="border-none"
                    value={field.state.value}
                    onChange={(val: TContentVariety) => field.handleChange(val)}
                    className="flex-shrink-0"
                    placement="bottom-end"
                  >
                    {CONTENT_VARIETY.map((variety) => (
                      <CustomSelect.Option key={variety.key} value={variety.key}>
                        {variety.label}
                      </CustomSelect.Option>
                    ))}
                  </CustomSelect>
                )}
              </form.Field>
            </div>
            {isPDFSelected && (
              <div className="flex items-center justify-between gap-2">
                <h6 className="flex-shrink-0 text-13 text-secondary">Page format</h6>
                <form.Field name="page_format">
                  {(field) => (
                    <CustomSelect
                      label={PAGE_FORMATS.find((format) => format.key === field.state.value)?.label}
                      disabled={isExporting}
                      buttonClassName="border-none"
                      value={field.state.value}
                      onChange={(val: TPageFormats) => field.handleChange(val)}
                      className="flex-shrink-0"
                      placement="bottom-end"
                    >
                      {PAGE_FORMATS.map((format) => (
                        <CustomSelect.Option key={format.key.toString()} value={format.key}>
                          {format.label}
                        </CustomSelect.Option>
                      ))}
                    </CustomSelect>
                  )}
                </form.Field>
              </div>
            )}
          </div>
          {(!memberRosterQuery.isSuccess || memberRosterIsError) && (
            <div role={memberRosterIsError ? "alert" : "status"} className="mt-3 text-13 text-tertiary">
              {memberRosterIsError
                ? "Workspace members could not be loaded. Retry before exporting to preserve page mentions."
                : "Loading workspace members before export…"}
              {memberRosterIsError && (
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  disabled={memberRosterIsLoading}
                  onClick={() => void retryMemberRoster()}
                >
                  {memberRosterIsLoading ? "Retrying" : "Retry"}
                </Button>
              )}
            </div>
          )}
        </div>
        <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
          <Button type="button" disabled={isExporting} variant="secondary" size="lg" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            size="lg"
            loading={isExporting}
            disabled={!memberRosterQuery.isSuccess || memberRosterIsError}
            type="submit"
          >
            {isExporting ? "Exporting" : "Export"}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
