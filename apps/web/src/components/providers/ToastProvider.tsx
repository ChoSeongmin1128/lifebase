"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { CheckCircle2, Info, TriangleAlert, X, XCircle } from "lucide-react";

type ToastVariant = "success" | "info" | "warning" | "error";

interface ToastOptions {
  title?: string;
  description?: string;
  variant?: ToastVariant;
  duration?: number;
  actionLabel?: string;
  onAction?: () => void;
}

interface ToastRecord extends Required<Pick<ToastOptions, "variant">> {
  id: number;
  title?: string;
  description?: string;
  duration: number;
  actionLabel?: string;
  onAction?: () => void;
}

interface ToastApi {
  show: (options: ToastOptions) => void;
  success: (title?: string, description?: string) => void;
  info: (title?: string, description?: string) => void;
  warning: (title?: string, description?: string) => void;
  error: (title?: string, description?: string) => void;
}

const ToastContext = createContext<ToastApi | null>(null);

const DEFAULT_DURATION: Record<ToastVariant, number> = {
  success: 3000,
  info: 3500,
  warning: 5000,
  error: 8000,
};

const DEDUPE_WINDOW_MS = 1500;

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastRecord[]>([]);
  const nextID = useRef(1);
  const dedupe = useRef<Map<string, number>>(new Map());
  const timers = useRef<Map<number, number>>(new Map());
  const stackRef = useRef<HTMLDivElement>(null);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((item) => item.id !== id));
    const timerID = timers.current.get(id);
    if (timerID !== undefined) {
      window.clearTimeout(timerID);
      timers.current.delete(id);
    }
  }, []);

  const show = useCallback((options: ToastOptions) => {
    const variant = options.variant ?? "info";
    const key = `${variant}:${options.title ?? ""}:${options.description ?? ""}:${options.actionLabel ?? ""}`;
    const now = Date.now();
    const last = dedupe.current.get(key) ?? 0;
    if (now-last < DEDUPE_WINDOW_MS) {
      return;
    }
    dedupe.current.set(key, now);

    const id = nextID.current++;
    const duration = options.duration ?? DEFAULT_DURATION[variant];
    const next: ToastRecord = {
      id,
      title: options.title,
      description: options.description,
      variant,
      duration,
      actionLabel: options.actionLabel,
      onAction: options.onAction,
    };
    setToasts((prev) => [next, ...prev].slice(0, 4));

    if (duration > 0) {
      const timerID = window.setTimeout(() => {
        dismiss(id);
      }, duration);
      timers.current.set(id, timerID);
    }
  }, [dismiss]);

  const api = useMemo<ToastApi>(() => ({
    show,
    success: (title, description) => show({ variant: "success", title, description }),
    info: (title, description) => show({ variant: "info", title, description }),
    warning: (title, description) => show({ variant: "warning", title, description }),
    error: (title, description) => show({ variant: "error", title, description }),
  }), [show]);

  useEffect(() => {
    const stack = stackRef.current;
    if (!stack) return;

    const root = document.documentElement;
    const updateOffset = () => {
      root.style.setProperty("--lb-toast-stack-height", `${stack.offsetHeight}px`);
    };

    updateOffset();

    if (typeof ResizeObserver === "undefined") {
      return () => {
        root.style.removeProperty("--lb-toast-stack-height");
      };
    }

    const observer = new ResizeObserver(updateOffset);
    observer.observe(stack);
    return () => {
      observer.disconnect();
      root.style.removeProperty("--lb-toast-stack-height");
    };
  }, [toasts]);

  const getVariantClass = (variant: ToastVariant) => {
    switch (variant) {
      case "success":
        return "border-success/40";
      case "warning":
        return "border-warning/40";
      case "error":
        return "border-error/40";
      case "info":
      default:
        return "border-info/40";
    }
  };

  const getVariantIcon = (variant: ToastVariant) => {
    switch (variant) {
      case "success":
        return <CheckCircle2 size={16} className="mt-0.5 shrink-0 text-success" />;
      case "warning":
        return <TriangleAlert size={16} className="mt-0.5 shrink-0 text-warning" />;
      case "error":
        return <XCircle size={16} className="mt-0.5 shrink-0 text-error" />;
      case "info":
      default:
        return <Info size={16} className="mt-0.5 shrink-0 text-info" />;
    }
  };

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div
        ref={stackRef}
        className="pointer-events-none fixed bottom-4 right-4 z-[120] flex w-[min(92vw,360px)] flex-col gap-2 max-sm:left-1/2 max-sm:right-auto max-sm:-translate-x-1/2"
      >
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`pointer-events-auto rounded-lg border bg-surface p-3 shadow-lg ${getVariantClass(toast.variant)}`}
            role="status"
            aria-live={toast.variant === "error" ? "assertive" : "polite"}
          >
            <div className="flex items-start gap-2">
              {getVariantIcon(toast.variant)}
              <div className="min-w-0 flex-1">
                {toast.title ? (
                  <p className="text-sm font-medium text-text-strong">{toast.title}</p>
                ) : null}
                {toast.description ? (
                  <p className={`${toast.title ? "mt-0.5 " : ""}text-xs text-text-secondary`}>{toast.description}</p>
                ) : null}
                {toast.actionLabel && toast.onAction ? (
                  <button
                    type="button"
                    onClick={() => {
                      toast.onAction?.();
                      dismiss(toast.id);
                    }}
                    className={`${toast.title || toast.description ? "mt-2 " : ""}text-xs font-medium text-primary hover:underline`}
                  >
                    {toast.actionLabel}
                  </button>
                ) : null}
              </div>
              <button
                type="button"
                onClick={() => dismiss(toast.id)}
                className="rounded p-1 text-text-muted hover:bg-surface-accent hover:text-text-primary"
                aria-label="닫기"
              >
                <X size={14} />
              </button>
            </div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return context;
}
