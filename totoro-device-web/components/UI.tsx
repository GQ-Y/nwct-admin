import React, { useState, useRef, useEffect } from 'react';
import { Search, ChevronDown, ChevronUp, Check, ChevronLeft, ChevronRight, Eye, EyeOff } from 'lucide-react';

export const Card: React.FC<React.HTMLAttributes<HTMLDivElement> & { title?: React.ReactNode; extra?: React.ReactNode; glass?: boolean }> = ({ title, children, className = '', extra, glass, ...props }) => (
  <div className={`card ${glass ? 'glass' : ''} ${className}`} {...props}>
    {title && <div className="card-title"><span>{title}</span>{extra && <div>{extra}</div>}</div>}
    {children}
  </div>
);

export const Button: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'outline' | 'ghost' | 'danger' }> = ({ variant = 'primary', className = '', ...props }) => {
  let btnClass = `btn btn-${variant}`;
  if (variant === 'danger') {
    // Custom danger variant styling logic usually handled in CSS, but here we can rely on `btn-danger` if defined, 
    // or we can inline style or class. For now we follow the pattern. 
    // Note: CSS doesn't have .btn-danger yet based on read file, but we can add or assume existing pattern.
    // However, the existing css uses .btn-outline for reset but red color. 
    // We will stick to the provided variants in CSS or just use standard class names.
    // If 'danger' is passed, it might default to nothing or we can handle it.
    // Let's use 'outline' + custom style if needed, but user asked for functionality.
    // Actually, looking at System.tsx, it uses variant="outline" style={{ color: '#f5222d' }}.
    // We will keep it simple.
  }
  return <button className={`${btnClass} ${className}`} {...props} />;
};

export const Input: React.FC<React.InputHTMLAttributes<HTMLInputElement>> = (props) => {
  const [showPassword, setShowPassword] = useState(false);
  const isPassword = props.type === 'password';

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      <input 
        className="input" 
        {...props} 
        type={isPassword && showPassword ? 'text' : props.type}
        style={{ 
          ...props.style,
          paddingRight: isPassword ? '40px' : props.style?.paddingRight 
        }} 
      />
      {isPassword && (
        <button
          type="button"
          onClick={() => setShowPassword(!showPassword)}
          style={{
            position: 'absolute',
            right: 12,
            top: '50%',
            transform: 'translateY(-50%)',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            padding: 4,
            display: 'flex',
            alignItems: 'center',
            color: 'var(--text-secondary)'
          }}
        >
          {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
        </button>
      )}
    </div>
  );
};

export const SuffixInput: React.FC<{
  value: string;
  onChange: (v: string) => void;
  suffixText: string;
  placeholder?: string;
}> = ({ value, onChange, suffixText, placeholder }) => {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        width: "100%",
        gap: 10,
      }}
    >
      <input
        className="input"
        value={value}
        onChange={(e) => onChange((e.target as any).value)}
        placeholder={placeholder}
        style={{ flex: 1, minWidth: 0 }}
      />
      <div
        className="input"
        style={{
          width: "auto",
          flex: "0 0 auto",
          padding: "0 16px",
          height: "48px",
          display: "flex",
          alignItems: "center",
          color: "var(--text-secondary)",
          cursor: "default",
          userSelect: "none",
          pointerEvents: "none",
          borderColor: "transparent",
          boxShadow: "none",
          whiteSpace: "nowrap",
        }}
        title={suffixText}
      >
        {suffixText}
      </div>
    </div>
  );
};

export const SearchInput: React.FC<React.InputHTMLAttributes<HTMLInputElement>> = (props) => (
  <div style={{ position: 'relative', width: props.width || '100%' }}>
    <Search size={18} style={{ position: 'absolute', left: 16, top: '50%', transform: 'translateY(-50%)', color: 'var(--text-secondary)' }} />
    <input 
      className="input" 
      {...props} 
      style={{ 
        paddingLeft: 48, 
        borderRadius: '9999px',
        height: '48px',
        ...props.style 
      }} 
    />
  </div>
);

interface SelectOption {
  label: string;
  value: string;
}

interface SelectProps {
  options: SelectOption[];
  value?: string;
  onChange?: (value: string) => void;
  width?: string | number;
  placeholder?: string;
}

export const Select: React.FC<SelectProps> = ({ options, value, onChange, width = 'auto', placeholder = 'Select...' }) => {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const selectedOption = options.find(opt => opt.value === value);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSelect = (optionValue: string) => {
    onChange?.(optionValue);
    setIsOpen(false);
  };

  return (
    <div ref={containerRef} style={{ position: 'relative', width, minWidth: 160 }}>
      <div 
        className="input"
        onClick={() => setIsOpen(!isOpen)}
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          cursor: 'pointer',
          height: '48px',
          padding: '0 20px',
          borderRadius: '16px',
          backgroundColor: isOpen ? '#FFFFFF' : 'var(--bg-input)',
          boxShadow: isOpen ? '0 0 0 2px rgba(10, 89, 247, 0.1)' : 'none',
          borderColor: isOpen ? 'var(--primary)' : 'transparent',
          color: 'var(--text-primary)'
        }}
      >
        <span style={{ color: selectedOption ? 'inherit' : 'var(--text-secondary)' }}>
          {selectedOption ? selectedOption.label : placeholder}
        </span>
        {isOpen ? <ChevronUp size={18} color="var(--primary)" /> : <ChevronDown size={18} color="var(--text-secondary)" />}
      </div>

      {isOpen && (
        <div style={{
          position: 'absolute',
          top: 'calc(100% + 8px)',
          left: 0,
          right: 0,
          background: 'rgba(255, 255, 255, 0.95)',
          backdropFilter: 'blur(16px)',
          borderRadius: '16px',
          boxShadow: '0 10px 40px rgba(0, 0, 0, 0.12)',
          zIndex: 100,
          padding: '8px',
          maxHeight: '240px',
          overflowY: 'auto',
          border: '1px solid rgba(255, 255, 255, 0.6)'
        }}>
          {options.map((option) => (
            <div
              key={option.value}
              onClick={() => handleSelect(option.value)}
              style={{
                padding: '12px 16px',
                borderRadius: '12px',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                color: option.value === value ? 'var(--primary)' : 'var(--text-primary)',
                background: option.value === value ? 'rgba(10, 89, 247, 0.08)' : 'transparent',
                transition: 'all 0.2s',
                marginBottom: '4px',
                fontWeight: option.value === value ? 600 : 400
              }}
              onMouseEnter={(e) => {
                if (option.value !== value) e.currentTarget.style.background = 'rgba(0, 0, 0, 0.04)';
              }}
              onMouseLeave={(e) => {
                if (option.value !== value) e.currentTarget.style.background = 'transparent';
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

export const Badge: React.FC<{ status: 'online' | 'offline' | 'warn' | 'error' | 'success'; text?: string }> = ({ status, text }) => {
  let type = 'success';
  if (status === 'offline' || status === 'error') type = 'error';
  if (status === 'warn') type = 'warning';
  
  return <span className={`badge badge-${type}`}>{text || status}</span>;
};

export const ProgressBar: React.FC<{ value: number; color?: string }> = ({ value, color }) => (
  <div className="progress-bg">
    <div className="progress-bar" style={{ width: `${value}%`, background: color }} />
  </div>
);

export const Alert: React.FC<{ type?: 'info' | 'error'; children: React.ReactNode }> = ({ type = 'info', children }) => (
  <div style={{ 
    padding: '16px', 
    background: type === 'error' ? 'rgba(232, 64, 38, 0.05)' : 'rgba(10, 89, 247, 0.05)', 
    border: `1px solid ${type === 'error' ? 'rgba(232, 64, 38, 0.2)' : 'rgba(10, 89, 247, 0.2)'}`, 
    borderRadius: '16px', 
    marginBottom: '24px', 
    color: type === 'error' ? 'var(--error)' : 'var(--primary)',
    display: 'flex',
    alignItems: 'center',
    gap: '12px'
  }}>
    {children}
  </div>
);

export const Pagination: React.FC<{
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  texts?: { prev: string; next: string; info?: string };
}> = ({ currentPage, totalPages, onPageChange, texts }) => (
  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 16, marginTop: 24 }}>
    <Button 
      variant="outline" 
      onClick={() => onPageChange(Math.max(1, currentPage - 1))}
      disabled={currentPage === 1}
      style={{ display: 'flex', alignItems: 'center', gap: 8, paddingLeft: 12, paddingRight: 16 }}
    >
      <ChevronLeft size={16} />
      {texts?.prev || 'Previous'}
    </Button>
    
    <div style={{ fontSize: 14, color: 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
      {texts?.info || `Page ${currentPage} of ${totalPages}`}
    </div>

    <Button 
      variant="outline" 
      onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
      disabled={currentPage === totalPages}
      style={{ display: 'flex', alignItems: 'center', gap: 8, paddingLeft: 16, paddingRight: 12 }}
    >
      {texts?.next || 'Next'}
      <ChevronRight size={16} />
    </Button>
  </div>
);
