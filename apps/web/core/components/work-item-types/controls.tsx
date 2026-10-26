import { typeErrorMessage } from "@/services/work-item-types.service";
import { Dialog, Listbox, ListboxButton, ListboxOptions, ListboxOption } from "@headlessui/react";
import {
  Check,
  Layers,
  ChevronDown,
  X,
  Hash,
  Type,
  Calendar,
  Users,
  ListFilter,
  ToggleLeft,
  Link,
  FileText,
  FunctionSquare,
  GitBranch,
  Package,
} from "lucide-react";
import type { ReactNode } from "react";
import { useId } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { Button } from "@plane/propel/button";
import { EModalWidth, ModalCore } from "@plane/ui";
import { typeIconComponents } from "./type-icons";
import type { PreviewType, PropertyKind } from "./preview-content";

export const inputStyle =
  "w-full rounded-md border border-strong bg-surface-1 px-3 py-2 text-13 text-primary outline-none focus:border-accent-strong focus:ring-1 focus:ring-accent-strong disabled:bg-layer-1 disabled:text-tertiary";
export function TypeIcon({ type, size = "md" }: { type: Pick<PreviewType, "icon" | "color">; size?: "sm" | "md" }) {
  const Icon = typeIconComponents[type.icon] ?? typeIconComponents.task;
  return (
    <span
      className={`inline-flex shrink-0 items-center justify-center rounded-md ${size === "sm" ? "size-5" : "size-7"}`}
      style={{ color: type.color, backgroundColor: `${type.color}18` }}
    >
      <Icon className={size === "sm" ? "size-3.5" : "size-4"} aria-hidden />
    </span>
  );
}
export function PropertyIcon({ kind }: { kind: PropertyKind }) {
  const Icon = {
    Text: Type,
    Number: Hash,
    Dropdown: ListFilter,
    Boolean: ToggleLeft,
    Date: Calendar,
    "Member picker": Users,
    "Release picker": Package,
    "Rich text": FileText,
    URL: Link,
    Formula: FunctionSquare,
    Cascading: GitBranch,
  }[kind];
  return <Icon className="size-4 shrink-0 text-tertiary" aria-hidden />;
}
export function Field({
  label,
  children,
  hint,
}: {
  label: string;
  children: (id: string) => ReactNode;
  hint?: string;
}) {
  const id = useId();
  return (
    <div className="space-y-2">
      <label htmlFor={id} className="block text-13 font-medium text-primary">
        {label}
      </label>
      {children(id)}
      {hint && <p className="text-12 text-tertiary">{hint}</p>}
    </div>
  );
}
export function Choice({
  label,
  value,
  options,
  onChange,
  disabled,
}: {
  label: string;
  value: string;
  options: string[];
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  return (
    <Listbox value={value} onChange={onChange} disabled={disabled}>
      <ListboxButton aria-label={label} className={`${inputStyle} flex items-center justify-between gap-2`}>
        {value || label}
        <ChevronDown className="size-3" />
      </ListboxButton>
      <ListboxOptions
        anchor="bottom start"
        portal
        modal={false}
        className="z-50 mt-1 min-w-48 rounded-md border border-subtle bg-surface-1 p-1 text-13 shadow-raised-200 outline-none"
      >
        {options.map((option) => (
          <ListboxOption
            key={option}
            value={option}
            className="flex cursor-pointer items-center justify-between gap-3 rounded px-2 py-1.5 data-focus:bg-layer-2"
          >
            {({ selected }) => (
              <>
                {option}
                {selected && <Check className="size-3" />}
              </>
            )}
          </ListboxOption>
        ))}
      </ListboxOptions>
    </Listbox>
  );
}
export function PreviewDialog({
  title,
  description,
  children,
  footer,
  onClose,
  wide = false,
  dismissDisabled = false,
}: {
  title: string;
  description?: string;
  children: ReactNode;
  footer: ReactNode;
  onClose: () => void;
  wide?: boolean;
  dismissDisabled?: boolean;
}) {
  return (
    <ModalCore
      isOpen
      handleClose={() => {
        if (!dismissDisabled) onClose();
      }}
      width={wide ? EModalWidth.XXXL : EModalWidth.XL}
    >
      <div className="flex max-h-[85dvh] flex-col">
        <div className="flex items-start justify-between gap-4 px-6 pt-6 pb-4">
          <div>
            <Dialog.Title className="text-18 font-semibold text-primary">{title}</Dialog.Title>
            {description && (
              <Dialog.Description className="mt-1 text-13 text-tertiary">{description}</Dialog.Description>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={dismissDisabled}
            aria-label="Close dialog"
            className="rounded p-1 text-tertiary hover:bg-layer-2"
          >
            <X className="size-4" />
          </button>
        </div>
        <div className="min-h-0 overflow-y-auto px-6 pb-6">{children}</div>
        <div className="flex items-center justify-end gap-2 border-t border-subtle px-6 py-4">{footer}</div>
      </div>
    </ModalCore>
  );
}
export type ConfirmationValues = { confirmation: string; permanent: boolean };
type ConfirmationFormApi = ReturnType<typeof useConfirmationForm>;

function useConfirmationForm({
  token,
  tokenRequiredWhen,
  initialValues,
  onConfirm,
}: {
  token?: string;
  tokenRequiredWhen?: (values: ConfirmationValues) => boolean;
  initialValues?: Partial<ConfirmationValues>;
  onConfirm: (values: ConfirmationValues) => void | Promise<void>;
}) {
  return useForm({
    defaultValues: { confirmation: initialValues?.confirmation ?? "", permanent: initialValues?.permanent ?? false },
    validators: {
      onSubmit: ({ value }): string | undefined =>
        token && (!tokenRequiredWhen || tokenRequiredWhen(value)) && value.confirmation !== token
          ? "Confirmation does not match."
          : undefined,
    },
    onSubmit: async ({ value, formApi }) => {
      try {
        await onConfirm(value);
      } catch (cause) {
        formApi.setErrorMap({ onSubmit: typeErrorMessage(cause) });
        throw cause;
      }
    },
  });
}

export function Confirmation({
  title,
  children,
  action = "Confirm",
  token,
  tokenRequiredWhen,
  initialValues,
  renderFields,
  onConfirm,
  onClose,
}: {
  title: string;
  children: ReactNode;
  action?: string;
  token?: string;
  tokenRequiredWhen?: (values: ConfirmationValues) => boolean;
  initialValues?: Partial<ConfirmationValues>;
  renderFields?: (form: ConfirmationFormApi) => ReactNode;
  onConfirm: (values: ConfirmationValues) => void | Promise<void>;
  onClose: () => void;
}) {
  const formId = useId();
  const form = useConfirmationForm({ token, tokenRequiredWhen, initialValues, onConfirm });
  const pending = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const values = useStore(form.store, (state) => state.values);
  const requiresToken = !!token && (!tokenRequiredWhen || tokenRequiredWhen(values));
  return (
    <PreviewDialog
      title={title}
      onClose={() => {
        if (!pending) onClose();
      }}
      footer={
        <>
          <Button type="button" variant="secondary" disabled={pending} onClick={() => !pending && onClose()}>
            Cancel
          </Button>
          <Button type="submit" form={formId} variant="error-fill" disabled={pending}>
            {action}
          </Button>
        </>
      }
    >
      <form
        id={formId}
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => undefined);
        }}
        className="space-y-5 text-13 text-secondary"
      >
        {children}
        {error && (
          <p role="alert" className="text-danger-primary">
            {typeof error === "string" ? error : typeErrorMessage(error)}
          </p>
        )}
        {requiresToken && token && (
          <Field label={`Type ${token} to continue`}>
            {(id) => (
              <form.Field name="confirmation">
                {(field) => (
                  <input
                    id={id}
                    className={inputStyle}
                    value={field.state.value}
                    disabled={pending}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                    autoComplete="off"
                  />
                )}
              </form.Field>
            )}
          </Field>
        )}
        {renderFields?.(form)}
      </form>
    </PreviewDialog>
  );
}
export function EmptyState({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children?: ReactNode;
}) {
  return (
    <div className="rounded-lg border border-dashed border-strong px-6 py-12 text-center">
      <Layers className="mx-auto mb-3 size-7 text-tertiary" />
      <h3 className="text-15 font-medium text-primary">{title}</h3>
      <p className="mx-auto mt-2 mb-5 max-w-md text-13 text-tertiary">{description}</p>
      {children}
    </div>
  );
}
