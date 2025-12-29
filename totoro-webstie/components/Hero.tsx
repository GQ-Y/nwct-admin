
import React, { useState, useEffect, useRef } from 'react';
import { Language, ProductType } from '../types';
import { TRANSLATIONS } from '../constants';
import { ArrowRight, Cpu, Network, Shield, Activity, Wifi } from 'lucide-react';

interface HeroProps {
  lang: Language;
  onNavigate: (view: 'home' | ProductType) => void;
}

const Hero: React.FC<HeroProps> = ({ lang, onNavigate }) => {
  const t = TRANSLATIONS[lang].hero;
  const cardRef = useRef<HTMLDivElement>(null);
  const [rotation, setRotation] = useState({ x: 0, y: 0 });
  const [logs, setLogs] = useState<string[]>([]);

  // Mouse Tilt Effect
  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!cardRef.current) return;

    const rect = cardRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    
    const centerX = rect.width / 2;
    const centerY = rect.height / 2;

    const rotateX = ((y - centerY) / centerY) * -5; // Max 5 deg rotation
    const rotateY = ((x - centerX) / centerX) * 5;

    setRotation({ x: rotateX, y: rotateY });
  };

  const handleMouseLeave = () => {
    setRotation({ x: 0, y: 0 });
  };

  // Terminal Log Simulation
  useEffect(() => {
    const bootSequence = [
      "Init Totoro Kernel...",
      "Loading Drivers... [OK]",
      "ETH0: Link UP 100Mbps",
      "Crypto Module: AES-256",
      "Bridge Connected.",
      "System Online."
    ];
    let index = 0;

    const interval = setInterval(() => {
      setLogs(prev => {
        const newLogs = [...prev, bootSequence[index % bootSequence.length]];
        if (newLogs.length > 5) newLogs.shift();
        return newLogs;
      });
      index++;
    }, 800);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="relative min-h-screen flex flex-col pt-16 overflow-hidden">
      {/* Background Effects */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] bg-brand-green/5 blur-[120px] rounded-full pointer-events-none animate-pulse-slow"></div>
      
      <main className="flex-grow flex flex-col items-center px-4 sm:px-6 lg:px-8 relative z-10 text-center pt-8 md:pt-16 md:justify-center">
        
        {/* Badge */}
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-brand-green/30 bg-brand-green/5 text-brand-green text-xs font-mono tracking-wider uppercase mb-6 md:mb-8">
          <span className="w-2 h-2 rounded-full bg-brand-green animate-pulse"></span>
          {t.badge}
        </div>

        {/* Title */}
        <h1 className="text-5xl md:text-7xl lg:text-8xl font-bold tracking-tighter text-white mb-4 md:mb-6 max-w-5xl mx-auto leading-tight">
          {t.title}
        </h1>

        {/* Subtitle */}
        <p className="text-lg md:text-xl text-gray-400 max-w-2xl mx-auto mb-10 font-light leading-relaxed">
          {t.subtitle}
        </p>

        {/* Buttons */}
        <div className="flex flex-col sm:flex-row gap-4 w-full sm:w-auto">
          <button 
            onClick={() => onNavigate('ultra')}
            className="group bg-white text-black px-8 py-3 rounded-lg font-bold flex items-center justify-center gap-2 hover:bg-gray-200 transition-all"
          >
            {t.ctaPrimary}
            <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
          </button>
          <button 
            className="px-8 py-3 rounded-lg font-medium border border-white/20 hover:bg-white/5 text-white transition-all flex items-center justify-center gap-2"
            onClick={() => window.open('https://github.com', '_blank')}
          >
            {t.ctaSecondary}
          </button>
        </div>

        {/* Visual / 3D Board Simulation */}
        <div 
            className="mt-20 relative w-full max-w-4xl aspect-[16/9] perspective-1000"
            onMouseMove={handleMouseMove}
            onMouseLeave={handleMouseLeave}
        >
             {/* Abstract Board Representation */}
            <div 
                ref={cardRef}
                className="w-full h-full bg-[#121212] border border-white/10 rounded-t-2xl shadow-2xl relative overflow-hidden flex flex-col items-center justify-center group transition-transform duration-100 ease-linear"
                style={{
                    transform: `perspective(1000px) rotateX(${rotation.x}deg) rotateY(${rotation.y}deg)`,
                    transformStyle: 'preserve-3d'
                }}
            >
                 <div className="absolute inset-0 bg-gradient-to-b from-brand-blue/5 to-transparent pointer-events-none"></div>
                 
                 {/* Moving Scanning Line */}
                 <div className="absolute top-0 left-0 w-full h-[2px] bg-brand-green/50 shadow-[0_0_15px_#00C853] animate-[scan_4s_ease-in-out_infinite] opacity-50 z-20"></div>

                 {/* Circuit Patterns (CSS Background) */}
                 <div className="absolute inset-0 opacity-10" style={{ 
                     backgroundImage: 'linear-gradient(rgba(255, 255, 255, 0.05) 1px, transparent 1px), linear-gradient(90deg, rgba(255, 255, 255, 0.05) 1px, transparent 1px)', 
                     backgroundSize: '40px 40px' 
                 }}></div>
                 
                 {/* Central Core Chip */}
                 <div className="z-10 relative flex flex-col items-center transform translate-z-20">
                     <div className="w-32 h-32 bg-black border border-white/20 rounded-xl flex items-center justify-center relative shadow-[0_0_30px_rgba(0,200,83,0.1)] group-hover:shadow-[0_0_50px_rgba(0,200,83,0.2)] transition-shadow duration-500">
                        <Cpu className="w-16 h-16 text-gray-500 group-hover:text-brand-green transition-colors duration-500" />
                        {/* Chip Pins */}
                        <div className="absolute -left-2 top-1/2 -translate-y-1/2 w-2 h-20 bg-repeat-y bg-[length:100%_8px] opacity-50 from-gray-700 to-transparent" style={{backgroundImage: 'linear-gradient(to bottom, #333 4px, transparent 4px)'}}></div>
                        <div className="absolute -right-2 top-1/2 -translate-y-1/2 w-2 h-20 bg-repeat-y bg-[length:100%_8px] opacity-50 from-gray-700 to-transparent" style={{backgroundImage: 'linear-gradient(to bottom, #333 4px, transparent 4px)'}}></div>
                        
                        {/* Status Dot */}
                        <div className="absolute top-2 right-2 w-2 h-2 bg-brand-green rounded-full animate-pulse"></div>
                     </div>
                     <div className="font-mono text-xs text-brand-green mt-4 tracking-[0.3em] font-bold">LUCKFOX CORE</div>
                 </div>

                 {/* Simulated Terminal Window (Left) */}
                 <div className="absolute left-8 bottom-8 w-64 h-32 bg-black/80 border border-white/10 rounded-lg p-3 font-mono text-[10px] text-gray-400 overflow-hidden hidden md:block backdrop-blur-sm">
                    <div className="flex items-center gap-2 mb-2 border-b border-white/5 pb-1">
                        <div className="w-2 h-2 rounded-full bg-red-500"></div>
                        <div className="w-2 h-2 rounded-full bg-yellow-500"></div>
                        <div className="w-2 h-2 rounded-full bg-green-500"></div>
                        <span className="ml-auto text-xs opacity-50">TERM_01</span>
                    </div>
                    <div className="flex flex-col gap-1">
                        {logs.map((log, i) => (
                            <div key={i} className="truncate">
                                <span className="text-brand-green mr-2">{'>'}</span>
                                {log}
                            </div>
                        ))}
                        <div className="w-2 h-4 bg-brand-green animate-pulse"></div>
                    </div>
                 </div>

                 {/* Status Indicators (Right) */}
                 <div className="absolute right-8 top-1/2 -translate-y-1/2 flex flex-col gap-4 hidden md:flex">
                     <div className="flex items-center gap-3">
                         <div className="w-8 h-8 rounded bg-white/5 flex items-center justify-center border border-white/10">
                            <Wifi size={14} className="text-brand-blue" />
                         </div>
                         <div className="text-left">
                             <div className="text-[10px] text-gray-500">SIGNAL</div>
                             <div className="text-xs font-mono text-white">STRONG</div>
                         </div>
                     </div>
                     <div className="flex items-center gap-3">
                         <div className="w-8 h-8 rounded bg-white/5 flex items-center justify-center border border-white/10">
                            <Activity size={14} className="text-purple-400" />
                         </div>
                         <div className="text-left">
                             <div className="text-[10px] text-gray-500">LOAD</div>
                             <div className="text-xs font-mono text-white">12%</div>
                         </div>
                     </div>
                 </div>

                 {/* Floating Particles */}
                 <div className="absolute w-1 h-1 bg-brand-blue rounded-full top-1/4 left-1/4 animate-ping opacity-40"></div>
                 <div className="absolute w-1 h-1 bg-brand-green rounded-full bottom-1/4 right-1/4 animate-ping opacity-40 delay-500"></div>
                 <div className="absolute w-1.5 h-1.5 bg-white rounded-full top-1/3 right-1/3 animate-ping opacity-20 delay-1000"></div>
            </div>
            
            {/* Reflection */}
            <div className="absolute -bottom-12 left-0 w-full h-24 bg-gradient-to-t from-transparent to-[#0a0a0a] z-20"></div>
        </div>

      </main>

      {/* Stats Strip */}
      <div className="border-y border-white/5 bg-black/20 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto px-4 py-8 grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="flex items-center gap-4 justify-center group cursor-default">
                <div className="p-3 bg-brand-blue/10 rounded-lg group-hover:bg-brand-blue/20 transition-colors">
                    <Network className="text-brand-blue w-6 h-6" />
                </div>
                <div className="text-left">
                    <div className="text-xl font-bold font-mono text-white group-hover:text-brand-blue transition-colors">Open Source</div>
                    <div className="text-xs text-gray-500 uppercase tracking-wider">Full Stack Access</div>
                </div>
            </div>
             <div className="flex items-center gap-4 justify-center group cursor-default">
                <div className="p-3 bg-brand-green/10 rounded-lg group-hover:bg-brand-green/20 transition-colors">
                    <Cpu className="text-brand-green w-6 h-6" />
                </div>
                <div className="text-left">
                    <div className="text-xl font-bold font-mono text-white group-hover:text-brand-green transition-colors">Luckfox</div>
                    <div className="text-xs text-gray-500 uppercase tracking-wider">Optimized Core</div>
                </div>
            </div>
             <div className="flex items-center gap-4 justify-center group cursor-default">
                <div className="p-3 bg-purple-500/10 rounded-lg group-hover:bg-purple-500/20 transition-colors">
                    <Shield className="text-purple-500 w-6 h-6" />
                </div>
                <div className="text-left">
                    <div className="text-xl font-bold font-mono text-white group-hover:text-purple-400 transition-colors">Private</div>
                    <div className="text-xs text-gray-500 uppercase tracking-wider">Secure Tunnel</div>
                </div>
            </div>
        </div>
      </div>
      
      <style>{`
        @keyframes scan {
            0% { top: 0%; opacity: 0; }
            10% { opacity: 0.5; }
            90% { opacity: 0.5; }
            100% { top: 100%; opacity: 0; }
        }
      `}</style>
    </div>
  );
};

export default Hero;
