import React, { useEffect } from "react";

type ToastType = "success" | "error" | "info";

export function Toast(props: {
  open: boolean;
  type?: ToastType;
  message: string;
  durationMs?: number;
  onClose: () => void;
  showConfirm?: boolean;
}) {
  const { open, type = "info", message, durationMs = 2200, onClose, showConfirm } = props;

  useEffect(() => {
    if (!open) return;
    if (showConfirm) return;
    const t = window.setTimeout(() => onClose(), durationMs);
    return () => window.clearTimeout(t);
  }, [open, durationMs, onClose, showConfirm]);

  if (!open) return null;

  const border =
    type === "success" ? "rgba(65,186,65,0.35)" : type === "error" ? "rgba(239,68,68,0.35)" : "rgba(10,89,247,0.35)";
  const accent = type === "success" ? "#41BA41" : type === "error" ? "#EF4444" : "var(--primary)";

  return (
    <div
      style={{
        position: "fixed",
        right: 18,
        bottom: 18,
        zIndex: 9999,
        maxWidth: 420,
        width: "calc(100% - 36px)",
        pointerEvents: "none",
      }}
    >
      <div
        className="card glass"
        style={{
          pointerEvents: "auto",
          padding: 14,
          borderRadius: 14,
          border: `1px solid ${border}`,
          boxShadow: "0 12px 40px rgba(0,0,0,0.10)",
          marginBottom: 0,
        }}
      >
        <div style={{ display: "flex", alignItems: "flex-start", gap: 10 }}>
          <div style={{ width: 8, height: 8, borderRadius: 999, background: accent, marginTop: 6, flex: "0 0 auto" }} />
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontWeight: 800, fontSize: 13, lineHeight: 1.35, wordBreak: "break-word" }}>{message}</div>
          </div>
          {showConfirm ? (
            <button className="btn btn-primary" onClick={onClose} style={{ height: 32, padding: "0 12px", flex: "0 0 auto" }}>
              确认
            </button>
          ) : (
            <button
              className="btn btn-ghost"
              onClick={onClose}
              style={{ width: 32, height: 32, padding: 0, borderRadius: "50%", flex: "0 0 auto" }}
              title="关闭"
            >
              ×
            </button>
          )}
        </div>
      </div>
    </div>
  );
}


