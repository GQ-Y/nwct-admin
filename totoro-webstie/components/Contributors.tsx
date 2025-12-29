
import React, { useMemo } from 'react';
import { Language } from '../types';
import { TRANSLATIONS } from '../constants';

interface ContributorsProps {
  lang: Language;
}

const Contributors: React.FC<ContributorsProps> = ({ lang }) => {
  const t = TRANSLATIONS[lang].footer; // Reusing the translation from footer context

  // Generate a large list of fake contributor names
  const contributors = useMemo(() => {
     const prefixes = ['hyper', 'super', 'dev', 'code', 'net', 'sys', 'root', 'admin', 'user', 'null', 'void', 'pixel', 'bit', 'byte', 'nano', 'mega', 'giga', 'terra', 'linux', 'unix', 'web', 'git', 'src', 'bin', 'opt', 'var', 'tmp', 'etc', 'usr', 'mnt', 'echo', 'cat', 'vim', 'ssh'];
     const suffixes = ['master', 'guru', 'ninja', 'wizard', 'coder', 'hacker', 'ops', 'admin', 'dev', 'bot', 'ai', 'lab', 'box', 'ctl', 'srv', 'x', 'z', 'q', 'k', 'os', 'io', 'sh', 'js', 'rs', 'py', 'go', 'c', 'cpp', 'hq', 'hub', 'flow'];
     
     const names = [];
     for(let i=0; i<200; i++) {
        const prefix = prefixes[Math.floor(Math.random() * prefixes.length)];
        const suffix = suffixes[Math.floor(Math.random() * suffixes.length)];
        const num = Math.random() > 0.6 ? Math.floor(Math.random() * 99) : '';
        names.push(`${prefix}${suffix}${num}`);
     }
     return names;
  }, []);

  // Split list into 3 chunks for different rows to avoid obvious repetition patterns
  const row1 = contributors.slice(0, 66);
  const row2 = contributors.slice(66, 132);
  const row3 = contributors.slice(132, 200);

  return (
    <section className="bg-[#050505] border-t border-white/5 py-16 overflow-hidden relative">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mb-10 text-center">
        <h3 className="text-sm font-mono uppercase tracking-[0.3em] text-gray-500">
            {t.thanks}
        </h3>
      </div>

      <div className="flex flex-col gap-6 relative">
         {/* Gradient Masks for fade effect on edges */}
         <div className="absolute left-0 top-0 bottom-0 w-24 md:w-48 bg-gradient-to-r from-[#050505] to-transparent z-20 pointer-events-none"></div>
         <div className="absolute right-0 top-0 bottom-0 w-24 md:w-48 bg-gradient-to-l from-[#050505] to-transparent z-20 pointer-events-none"></div>

         {/* Row 1: Left to Right, Slow */}
         <MarqueeRow items={row1} direction="normal" duration="120s" />
         
         {/* Row 2: Right to Left, Slower */}
         <MarqueeRow items={row2} direction="reverse" duration="150s" />

         {/* Row 3: Left to Right, Slowest */}
         <MarqueeRow items={row3} direction="normal" duration="180s" />
      </div>
    </section>
  );
};

const MarqueeRow = ({ items, direction, duration }: { items: string[], direction: 'normal' | 'reverse', duration: string }) => {
    return (
        <div className="flex overflow-hidden select-none group">
            <div 
                className="flex shrink-0 gap-8 items-center py-2"
                style={{
                    animation: `marquee ${duration} linear infinite`,
                    animationDirection: direction
                }}
            >
                {items.map((name, i) => (
                    <span key={`a-${i}`} className="text-gray-700 font-mono text-sm md:text-base hover:text-brand-green transition-colors cursor-crosshair opacity-60 hover:opacity-100">
                        @{name}
                    </span>
                ))}
            </div>
            {/* Duplicate for seamless loop */}
            <div 
                className="flex shrink-0 gap-8 items-center py-2 pl-8"
                style={{
                    animation: `marquee ${duration} linear infinite`,
                    animationDirection: direction
                }}
            >
                {items.map((name, i) => (
                    <span key={`b-${i}`} className="text-gray-700 font-mono text-sm md:text-base hover:text-brand-green transition-colors cursor-crosshair opacity-60 hover:opacity-100">
                        @{name}
                    </span>
                ))}
            </div>
            
            <style>{`
                @keyframes marquee {
                    0% { transform: translateX(0%); }
                    100% { transform: translateX(-100%); }
                }
                .group:hover div {
                    animation-play-state: paused;
                }
            `}</style>
        </div>
    );
}

export default Contributors;
