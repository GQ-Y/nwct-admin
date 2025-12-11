import React from 'react';

export const Card: React.FC<React.HTMLAttributes<HTMLDivElement> & { title?: React.ReactNode; extra?: React.ReactNode; glass?: boolean }> = ({ title, children, className = '', extra, glass, ...props }) => (
  <div className={`card ${glass ? 'glass' : ''} ${className}`} {...props}>
    {title && <div className="card-title"><span>{title}</span>{extra && <div>{extra}</div>}</div>}
    {children}
  </div>
);

export const Button: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'outline' | 'ghost' }> = ({ variant = 'primary', className = '', ...props }) => (
  <button className={`btn btn-${variant} ${className}`} {...props} />
);

export const Input: React.FC<React.InputHTMLAttributes<HTMLInputElement>> = (props) => (
  <input className="input" {...props} />
);

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
