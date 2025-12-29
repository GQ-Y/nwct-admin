import React from 'react';

interface TotoroLogoProps {
  className?: string;
  size?: number;
}

const TotoroLogo: React.FC<TotoroLogoProps> = ({ className = '', size = 24 }) => {
  return (
    <svg 
      xmlns="http://www.w3.org/2000/svg" 
      viewBox="0 0 1024 1024" 
      className={className}
      width={size}
      height={size}
    >
      <defs>
        <linearGradient id="totoro-bg" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#0A59F7"/>
          <stop offset="1" stopColor="#0042D0"/>
        </linearGradient>
        <filter id="totoro-softShadow" x="-20%" y="-20%" width="140%" height="140%">
          <feDropShadow dx="0" dy="18" stdDeviation="22" floodColor="#000000" floodOpacity="0.18"/>
        </filter>
      </defs>

      {/* Rounded background */}
      <rect x="80" y="80" width="864" height="864" rx="190" fill="url(#totoro-bg)"/>

      {/* Totoro silhouette */}
      <g filter="url(#totoro-softShadow)" fill="#FFFFFF">
        {/* Ears */}
        <path d="M332 342c-6-86 38-164 90-206 34-27 55-10 52 26-6 76-54 158-122 208-10 7-19 2-20-28z"/>
        <path d="M692 342c6-86-38-164-90-206-34-27-55-10-52 26 6 76 54 158 122 208 10 7 19 2 20-28z"/>

        {/* Body */}
        <path d="M512 262c-192 0-276 160-276 348 0 194 132 318 276 318s276-124 276-318c0-188-84-348-276-348z"/>

        {/* Eyes */}
        <circle cx="432" cy="488" r="22"/>
        <circle cx="592" cy="488" r="22"/>

        {/* Belly marks */}
        <g opacity="0.16">
          <path d="M512 556c-92 0-164 66-164 148 0 90 72 156 164 156s164-66 164-156c0-82-72-148-164-148z" fill="#000"/>
          <path d="M512 600c-66 0-118 46-118 104 0 64 52 112 118 112s118-48 118-112c0-58-52-104-118-104z" fill="#000"/>
        </g>
      </g>
    </svg>
  );
};

export default TotoroLogo;

