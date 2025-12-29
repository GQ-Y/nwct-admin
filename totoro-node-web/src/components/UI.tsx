import React, { useEffect, useRef, useState } from "react";
import { Check, ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Search } from "lucide-react";

export const Card: React.FC<
  React.HTMLAttributes<HTMLDivElement> & { title?: React.ReactNode; extra?: React.ReactNode; glass?: boolean }
> = ({ title, children, className = "", extra, glass, ...props }) => (
  <div className={`card ${glass ? "glass" : ""} ${className}`} {...props}>
    {title && (
      <div className="card-title">
        <span>{title}</span>
        {extra && <div>{extra}</div>}
      </div>
    )}
    {children}
  </div>
);

export const Button: React.FC<
  React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "primary" | "outline" | "ghost" }
> = ({ variant = "primary", className = "", ...props }) => {
  return <button className={`btn btn-${variant} ${className}`} {...props} />;
};

export const Input: React.FC<React.InputHTMLAttributes<HTMLInputElement>> = (props) => {
  return <input className="input" {...props} />;
};

export const SearchInput: React.FC<React.InputHTMLAttributes<HTMLInputElement>> = (props) => (
  <div style={{ position: "relative", width: (props as any).width || "100%" }}>
    <Search
      size={18}
      style={{
        position: "absolute",
        left: 16,
        top: "50%",
        transform: "translateY(-50%)",
        color: "var(--text-secondary)",
      }}
    />
    <input
      className="input"
      {...props}
      style={{
        paddingLeft: 48,
        borderRadius: "9999px",
        height: "44px",
        ...(props.style || {}),
      }}
    />
  </div>
);

interface SelectOption {
  label: string;
  value: string;
}

export const Select: React.FC<{
  options: SelectOption[];
  value?: string;
  onChange?: (value: string) => void;
  width?: string | number;
  placeholder?: string;
}> = ({ options, value, onChange, width = "auto", placeholder = "请选择" }) => {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const selectedOption = options.find((opt) => opt.value === value);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleSelect = (optionValue: string) => {
    onChange?.(optionValue);
    setIsOpen(false);
  };

  return (
    <div ref={containerRef} style={{ position: "relative", width, minWidth: 160 }}>
      <div
        className="input"
        onClick={() => setIsOpen(!isOpen)}
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          cursor: "pointer",
          height: "44px",
          padding: "0 16px",
          borderRadius: "16px",
          backgroundColor: isOpen ? "#FFFFFF" : "var(--bg-input)",
          boxShadow: isOpen ? "0 0 0 2px rgba(10, 89, 247, 0.1)" : "none",
          borderColor: isOpen ? "var(--primary)" : "transparent",
          color: "var(--text-primary)",
        }}
      >
        <span style={{ color: selectedOption ? "inherit" : "var(--text-secondary)" }}>
          {selectedOption ? selectedOption.label : placeholder}
        </span>
        {isOpen ? <ChevronUp size={18} color="var(--primary)" /> : <ChevronDown size={18} color="var(--text-secondary)" />}
      </div>

      {isOpen && (
        <div
          style={{
            position: "absolute",
            top: "calc(100% + 8px)",
            left: 0,
            right: 0,
            background: "rgba(255, 255, 255, 0.95)",
            backdropFilter: "blur(16px)",
            borderRadius: "16px",
            boxShadow: "0 10px 40px rgba(0, 0, 0, 0.12)",
            zIndex: 100,
            padding: "8px",
            maxHeight: "240px",
            overflowY: "auto",
            border: "1px solid rgba(255, 255, 255, 0.6)",
          }}
        >
          {options.map((option) => (
            <div
              key={option.value}
              onClick={() => handleSelect(option.value)}
              style={{
                padding: "12px 16px",
                borderRadius: "12px",
                cursor: "pointer",
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                color: option.value === value ? "var(--primary)" : "var(--text-primary)",
                background: option.value === value ? "rgba(10, 89, 247, 0.08)" : "transparent",
                transition: "all 0.2s",
                marginBottom: "4px",
                fontWeight: option.value === value ? 700 : 500,
              }}
              onMouseEnter={(e) => {
                if (option.value !== value) (e.currentTarget.style.background = "rgba(0, 0, 0, 0.04)");
              }}
              onMouseLeave={(e) => {
                if (option.value !== value) (e.currentTarget.style.background = "transparent");
              }}
            >
              {option.label}
              {option.value === value && <Check size={16} />}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export const Badge: React.FC<{ status: "online" | "offline" | "warn" | "error" | "success"; text?: string }> = ({
  status,
  text,
}) => {
  let type: "success" | "error" | "warning" = "success";
  if (status === "offline" || status === "error") type = "error";
  if (status === "warn") type = "warning";
  return <span className={`badge badge-${type}`}>{text || status}</span>;
};

export const Pagination: React.FC<{
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  texts?: { prev: string; next: string; info?: string };
}> = ({ currentPage, totalPages, onPageChange, texts }) => (
  <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: 16, marginTop: 18 }}>
    <Button
      variant="outline"
      onClick={() => onPageChange(Math.max(1, currentPage - 1))}
      disabled={currentPage === 1}
      style={{ display: "flex", alignItems: "center", gap: 8, paddingLeft: 12, paddingRight: 14 }}
    >
      <ChevronLeft size={16} />
      {texts?.prev || "上一页"}
    </Button>

    <div style={{ fontSize: 13, color: "var(--text-secondary)", fontVariantNumeric: "tabular-nums" }}>
      {texts?.info || `第 ${currentPage} / ${totalPages} 页`}
    </div>

    <Button
      variant="outline"
      onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
      disabled={currentPage === totalPages}
      style={{ display: "flex", alignItems: "center", gap: 8, paddingLeft: 14, paddingRight: 12 }}
    >
      {texts?.next || "下一页"}
      <ChevronRight size={16} />
    </Button>
  </div>
);

export const Checkbox: React.FC<{
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  disabled?: boolean;
}> = ({ checked, onChange, label, disabled = false }) => {
  return (
    <label
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        cursor: disabled ? "not-allowed" : "pointer",
        userSelect: "none",
        opacity: disabled ? 0.6 : 1,
      }}
    >
      <div
        onClick={() => !disabled && onChange(!checked)}
        style={{
          width: 20,
          height: 20,
          borderRadius: 6,
          border: `2px solid ${checked ? "var(--primary)" : "#d0d0d0"}`,
          background: checked ? "var(--primary)" : "transparent",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          transition: "all 0.2s",
          flexShrink: 0,
          position: "relative",
        }}
      >
        {checked && (
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M10 3L4.5 8.5L2 6"
              stroke="white"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        )}
      </div>
      {label && (
        <span style={{ fontSize: 14, fontWeight: 700, color: disabled ? "var(--text-secondary)" : "var(--text-primary)" }}>
          {label}
        </span>
      )}
    </label>
  );
};

export const ConfirmDialog: React.FC<{
  open: boolean;
  title?: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  onConfirm: () => void;
  onCancel: () => void;
  variant?: "danger" | "warning" | "info";
}> = ({ open, title, message, confirmText = "确认", cancelText = "取消", onConfirm, onCancel, variant = "info" }) => {
  if (!open) return null;

  const variantButtonStyles = {
    danger: { background: "var(--error)", boxShadow: "0 4px 12px rgba(232, 64, 38, 0.25)" },
    warning: { background: "var(--warning)", boxShadow: "0 4px 12px rgba(232, 166, 0, 0.25)" },
    info: { background: "var(--primary)", boxShadow: "0 4px 12px rgba(10, 89, 247, 0.25)" },
  };

  const buttonStyle = variantButtonStyles[variant];

  return (
    <div
      className="modal-overlay"
      onMouseDown={onCancel}
      style={{
        position: "fixed",
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: "rgba(0, 0, 0, 0.4)",
        backdropFilter: "blur(8px)",
        WebkitBackdropFilter: "blur(8px)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
        animation: "fadeIn 0.2s ease-out",
      }}
    >
      <div
        className="card glass"
        onMouseDown={(e) => e.stopPropagation()}
        style={{
          width: "90%",
          maxWidth: 420,
          padding: 28,
          borderRadius: "var(--border-radius-lg)",
          background: "rgba(255, 255, 255, 0.95)",
          backdropFilter: "blur(24px)",
          WebkitBackdropFilter: "blur(24px)",
          border: "1px solid rgba(255, 255, 255, 0.8)",
          boxShadow: "var(--shadow-lg)",
          animation: "slideUp 0.3s cubic-bezier(0.4, 0, 0.2, 1)",
        }}
      >
        {title && (
          <div
            style={{
              fontSize: 18,
              fontWeight: 800,
              marginBottom: 12,
              color: "var(--text-primary)",
              lineHeight: 1.4,
            }}
          >
            {title}
          </div>
        )}
        <div
          style={{
            fontSize: 14,
            color: "var(--text-secondary)",
            marginBottom: 24,
            lineHeight: 1.6,
            fontWeight: 500,
          }}
        >
          {message}
        </div>
        <div style={{ display: "flex", gap: 12, justifyContent: "flex-end" }}>
          <Button
            variant="outline"
            onClick={onCancel}
            style={{
              minWidth: 80,
            }}
          >
            {cancelText}
          </Button>
          <Button
            variant="primary"
            onClick={onConfirm}
            style={{
              ...buttonStyle,
              minWidth: 80,
            }}
          >
            {confirmText}
          </Button>
        </div>
      </div>
    </div>
  );
};

