import Link from "next/link";
import type { InputHTMLAttributes, ReactNode, TextareaHTMLAttributes } from "react";
import clsx from "clsx";

export function Button({
  children,
  className,
  variant = "primary",
  size = "md",
  loading = false,
  disabled,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger" | "board";
  size?: "sm" | "md";
  loading?: boolean;
}) {
  return (
    <button
      className={clsx(
        "inline-flex items-center justify-center rounded-md font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50",
        size === "sm" ? "px-2.5 py-1 text-xs" : "px-3.5 py-2 text-sm",
        variant === "primary" &&
          "bg-primary text-white shadow-sm hover:bg-[var(--primary-hover)]",
        variant === "secondary" &&
          "border border-border bg-surface text-foreground hover:bg-background",
        variant === "ghost" &&
          "text-muted hover:bg-black/5 hover:text-foreground",
        variant === "danger" &&
          "bg-[var(--danger)] text-white hover:bg-red-700",
        variant === "board" &&
          "bg-white/20 text-white backdrop-blur-sm hover:bg-white/30",
        className
      )}
      disabled={disabled || loading}
      {...props}
    >
      {loading && (
        <Spinner
          className={clsx(
            "mr-2",
            size === "sm" ? "h-3 w-3" : "h-4 w-4",
            variant === "primary" || variant === "danger" || variant === "board"
              ? "border-white/30 border-t-white"
              : undefined
          )}
        />
      )}
      {children}
    </button>
  );
}

export function Input({
  className,
  ...props
}: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={clsx(
        "w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground outline-none transition-shadow placeholder:text-muted/70 focus:border-primary focus:ring-2 focus:ring-primary/20",
        className
      )}
      {...props}
    />
  );
}

export function Textarea({
  className,
  ...props
}: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      className={clsx(
        "w-full resize-y rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground outline-none transition-shadow placeholder:text-muted/70 focus:border-primary focus:ring-2 focus:ring-primary/20",
        className
      )}
      {...props}
    />
  );
}

export function Card({
  children,
  className,
  hover = false,
}: {
  children: ReactNode;
  className?: string;
  hover?: boolean;
}) {
  return (
    <div
      className={clsx(
        "rounded-lg border border-border bg-surface p-4 shadow-sm",
        hover && "transition-shadow hover:shadow-md",
        className
      )}
    >
      {children}
    </div>
  );
}

export function FieldLabel({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <label className={clsx("mb-1.5 block text-sm font-medium text-muted", className)}>
      {children}
    </label>
  );
}

export function Badge({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <span
      className={clsx(
        "inline-flex items-center rounded-full bg-background px-2.5 py-0.5 text-xs font-medium capitalize text-muted",
        className
      )}
    >
      {children}
    </span>
  );
}

export function Avatar({
  name,
  size = "md",
  className,
}: {
  name: string;
  size?: "sm" | "md";
  className?: string;
}) {
  const colors = [
    "bg-blue-600",
    "bg-violet-600",
    "bg-emerald-600",
    "bg-amber-600",
    "bg-rose-600",
    "bg-cyan-600",
  ];
  const color = colors[name.charCodeAt(0) % colors.length];
  const sizeClass = size === "sm" ? "h-6 w-6 text-[10px]" : "h-8 w-8 text-xs";

  return (
    <span
      className={clsx(
        "inline-flex shrink-0 items-center justify-center rounded-full font-semibold text-white",
        color,
        sizeClass,
        className
      )}
      title={name}
    >
      {name.charAt(0).toUpperCase()}
    </span>
  );
}

export function Spinner({ className }: { className?: string }) {
  return (
    <div
      className={clsx(
        "h-5 w-5 animate-spin rounded-full border-2 border-border border-t-primary",
        className
      )}
      role="status"
      aria-label="Loading"
    />
  );
}

export function PageLoader({ label = "Loading..." }: { label?: string }) {
  return (
    <div className="flex min-h-[40vh] flex-col items-center justify-center gap-3">
      <Spinner />
      <p className="text-sm text-muted">{label}</p>
    </div>
  );
}

export function PageHeader({
  backHref,
  backLabel,
  title,
  subtitle,
  actions,
}: {
  backHref?: string;
  backLabel?: string;
  title: string;
  subtitle?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
      <div>
        {backHref && backLabel && (
          <Link
            href={backHref}
            className="mb-1 inline-block text-sm text-muted transition-colors hover:text-foreground"
          >
            ← {backLabel}
          </Link>
        )}
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">{title}</h1>
        {subtitle && <p className="mt-1 text-sm text-muted">{subtitle}</p>}
      </div>
      {actions}
    </div>
  );
}

export function AuthShell({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: ReactNode;
}) {
  return (
    <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center bg-background p-4">
      <div className="w-full max-w-md">
        <div className="mb-6 text-center">
          <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-lg font-bold text-white shadow-sm">
            T
          </div>
          <h1 className="text-xl font-semibold text-foreground">{title}</h1>
          {subtitle && <p className="mt-1 text-sm text-muted">{subtitle}</p>}
        </div>
        <Card className="shadow-md">{children}</Card>
      </div>
    </div>
  );
}

export function EmptyState({
  title,
  description,
}: {
  title: string;
  description?: string;
}) {
  return (
    <div className="rounded-lg border border-dashed border-border bg-surface/50 px-6 py-10 text-center">
      <p className="font-medium text-foreground">{title}</p>
      {description && <p className="mt-1 text-sm text-muted">{description}</p>}
    </div>
  );
}

export function Alert({
  children,
  variant = "error",
}: {
  children: ReactNode;
  variant?: "error" | "success" | "info";
}) {
  return (
    <p
      className={clsx(
        "rounded-md px-3 py-2 text-sm",
        variant === "error" && "bg-red-50 text-[var(--danger)]",
        variant === "success" && "bg-emerald-50 text-[var(--success)]",
        variant === "info" && "bg-blue-50 text-primary"
      )}
    >
      {children}
    </p>
  );
}
