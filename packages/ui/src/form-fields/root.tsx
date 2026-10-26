/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { cn } from "@plane/utils/common";

type FormProps = Omit<React.ComponentProps<"form">, "onSubmit"> & { onSubmit: () => Promise<void> };

export function Form({ onSubmit, ...props }: FormProps) {
  return (
    <form
      {...props}
      onSubmit={(event) => {
        event.preventDefault();
        event.stopPropagation();
        void onSubmit();
      }}
    />
  );
}

// Reusable Label Component
interface LabelProps {
  htmlFor: string;
  children: React.ReactNode;
  className?: string;
}

export function Label({ htmlFor, children, className }: LabelProps) {
  return (
    <label htmlFor={htmlFor} className={cn("block text-13 font-medium text-primary", className)}>
      {children}
    </label>
  );
}

// Reusable Form Field Component
interface FormFieldProps {
  label: string;
  htmlFor: string;
  children: React.ReactNode;
  className?: string;
  optional?: boolean;
}

export function FormField({ label, htmlFor, children, className, optional = false }: FormFieldProps) {
  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <Label htmlFor={htmlFor}>
        {label}
        {optional && <span className="text-13 text-placeholder"> (optional)</span>}
      </Label>
      {children}
    </div>
  );
}

// Reusable Validation Message Component
interface ValidationMessageProps {
  type: "error" | "success";
  message: string;
  className?: string;
}

export function ValidationMessage({ type, message, className }: ValidationMessageProps) {
  return (
    <p
      className={cn(
        "text-13",
        {
          "text-danger-primary": type === "error",
          "text-success-primary": type === "success",
        },
        className
      )}
    >
      {message}
    </p>
  );
}

export function FormErrors({ errors, className }: { errors: readonly (string | undefined)[]; className?: string }) {
  const messages = [...new Set(errors.filter((error): error is string => Boolean(error)))];
  if (!messages.length) return null;
  return (
    <div role="alert" className={cn("text-13 text-danger-primary", className)}>
      {messages.map((message) => (
        <p key={message}>{message}</p>
      ))}
    </div>
  );
}
