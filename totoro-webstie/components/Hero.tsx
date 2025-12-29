
import React, { useState, useEffect, useRef } from 'react';
import { Language, ProductType } from '../types';
import { TRANSLATIONS } from '../constants';
import { ArrowRight, Cpu, Network, Shield, Activity, Wifi, Sparkles } from 'lucide-react';
import AnimatedCounter from './AnimatedCounter';
import { useScrollAnimation } from '../hooks/useScrollAnimation';

interface HeroProps {
  lang: Language;
  onNavigate: (view: 'home' | ProductType) => void;
}

const Hero: React.FC<HeroProps> = ({ lang, onNavigate }) => {
  const t = TRANSLATIONS[lang].hero;
  const cardRef = useRef<HTMLDivElement>(null);
  const heroRef = useRef<HTMLDivElement>(null);
  const badgeRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const [rotation, setRotation] = useState({ x: 0, y: 0 });
  const [logs, setLogs] = useState<string[]>([]);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });
  const [isVisible, setIsVisible] = useState(false);
  const [particles, setParticles] = useState<Array<{ id: number; x: number; y: number; delay: number }>>([]);

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

  // Mouse position tracking for parallax
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMousePosition({
        x: (e.clientX / window.innerWidth - 0.5) * 20,
        y: (e.clientY / window.innerHeight - 0.5) * 20
      });
    };

    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  // Intersection Observer for scroll animations
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setIsVisible(true);
          }
        });
      },
      { threshold: 0.1 }
    );

    if (heroRef.current) {
      observer.observe(heroRef.current);
    }

    return () => {
      if (heroRef.current) {
        observer.unobserve(heroRef.current);
      }
    };
  }, []);

  // Generate floating particles
  useEffect(() => {
    const newParticles = Array.from({ length: 20 }, (_, i) => ({
      id: i,
      x: Math.random() * 100,
      y: Math.random() * 100,
      delay: Math.random() * 5
    }));
    setParticles(newParticles);
  }, []);

  return (
    <div ref={heroRef} className="relative min-h-screen flex flex-col pt-16 overflow-hidden">
      {/* Animated Background Particles */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        {particles.map((particle) => (
          <div
            key={particle.id}
            className="absolute w-1 h-1 bg-brand-green/30 rounded-full"
            style={{
              left: `${particle.x}%`,
              top: `${particle.y}%`,
              animation: `float ${3 + Math.random() * 2}s ease-in-out infinite`,
              animationDelay: `${particle.delay}s`,
              transform: `translate(${mousePosition.x * 0.1}px, ${mousePosition.y * 0.1}px)`
            }}
          />
        ))}
      </div>

      {/* Background Effects with Parallax */}
      <div 
        className="absolute top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] bg-brand-green/5 blur-[120px] rounded-full pointer-events-none animate-pulse-slow transition-transform duration-300"
        style={{
          transform: `translate(-50%, 0) translate(${mousePosition.x * 0.3}px, ${mousePosition.y * 0.3}px)`
        }}
      ></div>
      
      <main className="flex-grow flex flex-col items-center px-4 sm:px-6 lg:px-8 relative z-10 text-center pt-8 md:pt-16 md:justify-center">
        
        {/* Badge with fade-in animation */}
        <div 
          ref={badgeRef}
          className={`inline-flex items-center gap-2 px-3 py-1 rounded-full border border-brand-green/30 bg-brand-green/5 text-brand-green text-xs font-mono tracking-wider uppercase mb-6 md:mb-8 transition-all duration-1000 ${
            isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4'
          }`}
          style={{
            transitionDelay: '0.2s'
          }}
        >
          <span className="w-2 h-2 rounded-full bg-brand-green animate-pulse"></span>
          <Sparkles className="w-3 h-3 animate-spin-slow" />
          {t.badge}
        </div>

        {/* Title with fade-in and scale animation */}
        <h1 
          ref={titleRef}
          className={`text-5xl md:text-7xl lg:text-8xl font-bold tracking-tighter text-white mb-4 md:mb-6 max-w-5xl mx-auto leading-tight transition-all duration-1000 ${
            isVisible ? 'opacity-100 translate-y-0 scale-100' : 'opacity-0 translate-y-8 scale-95'
          }`}
          style={{
            transitionDelay: '0.4s',
            transform: `translateY(${isVisible ? 0 : 32}px) scale(${isVisible ? 1 : 0.95}) translate(${mousePosition.x * 0.05}px, ${mousePosition.y * 0.05}px)`
          }}
        >
          {t.title}
        </h1>

        {/* Subtitle with fade-in animation */}
        <p 
          ref={subtitleRef}
          className={`text-lg md:text-xl text-gray-400 max-w-2xl mx-auto mb-10 font-light leading-relaxed transition-all duration-1000 ${
            isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-6'
          }`}
          style={{
            transitionDelay: '0.6s'
          }}
        >
          {t.subtitle}
        </p>

        {/* Buttons with fade-in animation */}
        <div 
          className={`flex flex-col sm:flex-row gap-4 w-full sm:w-auto transition-all duration-1000 ${
            isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-6'
          }`}
          style={{
            transitionDelay: '0.8s'
          }}
        >
          <button 
            onClick={() => onNavigate('ultra')}
            className="group relative bg-white text-black px-8 py-3 rounded-lg font-bold flex items-center justify-center gap-2 hover:bg-gray-200 transition-all overflow-hidden hover:scale-105 active:scale-95"
          >
            <span className="absolute inset-0 bg-gradient-to-r from-brand-green/20 to-brand-blue/20 opacity-0 group-hover:opacity-100 transition-opacity"></span>
            <span className="relative z-10 flex items-center gap-2">
              {t.ctaPrimary}
              <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
            </span>
          </button>
          <button 
            className="group relative px-8 py-3 rounded-lg font-medium border border-white/20 hover:border-brand-green/50 hover:bg-white/5 text-white transition-all flex items-center justify-center gap-2 overflow-hidden hover:scale-105 active:scale-95"
            onClick={() => window.open('https://github.com', '_blank')}
          >
            <span className="absolute inset-0 bg-gradient-to-r from-transparent via-white/5 to-transparent translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-700"></span>
            <span className="relative z-10">{t.ctaSecondary}</span>
          </button>
        </div>

        {/* Visual / 3D Board Simulation */}
        <div 
            className={`mt-20 relative w-full max-w-4xl aspect-[16/9] perspective-1000 transition-all duration-1000 ${
              isVisible ? 'opacity-100 translate-y-0 scale-100' : 'opacity-0 translate-y-12 scale-95'
            }`}
            style={{ transitionDelay: '1s' }}
            onMouseMove={handleMouseMove}
            onMouseLeave={handleMouseLeave}
        >
             {/* Abstract Board Representation */}
            <div 
                ref={cardRef}
                className="w-full h-full bg-[#121212] border border-white/10 rounded-t-2xl shadow-2xl relative overflow-hidden flex flex-col items-center justify-center group transition-all duration-100 ease-linear hover:border-brand-green/30 hover:shadow-[0_0_40px_rgba(0,200,83,0.2)]"
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
                     <div className="w-32 h-32 bg-black border border-white/20 rounded-xl flex items-center justify-center relative shadow-[0_0_30px_rgba(0,200,83,0.1)] group-hover:shadow-[0_0_50px_rgba(0,200,83,0.2)] transition-all duration-500 group-hover:scale-110 group-hover:border-brand-green/50">
                        <Cpu className="w-16 h-16 text-gray-500 group-hover:text-brand-green transition-all duration-500 group-hover:scale-110 group-hover:rotate-12" />
                        {/* Chip Pins */}
                        <div className="absolute -left-2 top-1/2 -translate-y-1/2 w-2 h-20 bg-repeat-y bg-[length:100%_8px] opacity-50 from-gray-700 to-transparent group-hover:opacity-70 transition-opacity" style={{backgroundImage: 'linear-gradient(to bottom, #333 4px, transparent 4px)'}}></div>
                        <div className="absolute -right-2 top-1/2 -translate-y-1/2 w-2 h-20 bg-repeat-y bg-[length:100%_8px] opacity-50 from-gray-700 to-transparent group-hover:opacity-70 transition-opacity" style={{backgroundImage: 'linear-gradient(to bottom, #333 4px, transparent 4px)'}}></div>
                        
                        {/* Status Dot */}
                        <div className="absolute top-2 right-2 w-2 h-2 bg-brand-green rounded-full animate-pulse group-hover:scale-150 group-hover:shadow-[0_0_10px_#00C853] transition-all"></div>
                        
                        {/* Animated glow rings */}
                        <div className="absolute inset-0 rounded-xl border border-brand-green/20 opacity-0 group-hover:opacity-100 animate-ping"></div>
                     </div>
                     <div className="font-mono text-xs text-brand-green mt-4 tracking-[0.3em] font-bold group-hover:scale-110 transition-transform duration-300">LUCKFOX CORE</div>
                 </div>

                 {/* Simulated Terminal Window (Left) */}
                 <div className="absolute left-8 bottom-8 w-64 h-32 bg-black/80 border border-white/10 rounded-lg p-3 font-mono text-[10px] text-gray-400 overflow-hidden hidden md:block backdrop-blur-sm group-hover:border-brand-green/30 group-hover:shadow-[0_0_20px_rgba(0,200,83,0.2)] transition-all duration-300">
                    <div className="flex items-center gap-2 mb-2 border-b border-white/5 pb-1">
                        <div className="w-2 h-2 rounded-full bg-red-500 group-hover:animate-pulse"></div>
                        <div className="w-2 h-2 rounded-full bg-yellow-500 group-hover:animate-pulse" style={{ animationDelay: '0.2s' }}></div>
                        <div className="w-2 h-2 rounded-full bg-green-500 group-hover:animate-pulse" style={{ animationDelay: '0.4s' }}></div>
                        <span className="ml-auto text-xs opacity-50 group-hover:opacity-100 transition-opacity">TERM_01</span>
                    </div>
                    <div className="flex flex-col gap-1">
                        {logs.map((log, i) => (
                            <div 
                                key={i} 
                                className="truncate transition-all duration-300"
                                style={{ 
                                    animation: `fadeInUp 0.5s ease-out ${i * 0.1}s both`,
                                    opacity: isVisible ? 1 : 0
                                }}
                            >
                                <span className="text-brand-green mr-2">{'>'}</span>
                                {log}
                            </div>
                        ))}
                        <div className="w-2 h-4 bg-brand-green animate-pulse group-hover:shadow-[0_0_8px_#00C853]"></div>
                    </div>
                 </div>

                 {/* Status Indicators (Right) */}
                 <div className="absolute right-8 top-1/2 -translate-y-1/2 flex flex-col gap-4 hidden md:flex">
                     <div className="flex items-center gap-3 group/stat hover:scale-110 transition-transform duration-300">
                         <div className="w-8 h-8 rounded bg-white/5 flex items-center justify-center border border-white/10 group-hover/stat:border-brand-blue/50 group-hover/stat:bg-brand-blue/10 transition-all">
                            <Wifi size={14} className="text-brand-blue group-hover/stat:animate-pulse" />
                         </div>
                         <div className="text-left">
                             <div className="text-[10px] text-gray-500 group-hover/stat:text-brand-blue transition-colors">SIGNAL</div>
                             <div className="text-xs font-mono text-white group-hover/stat:text-brand-blue transition-colors">STRONG</div>
                         </div>
                     </div>
                     <div className="flex items-center gap-3 group/stat hover:scale-110 transition-transform duration-300">
                         <div className="w-8 h-8 rounded bg-white/5 flex items-center justify-center border border-white/10 group-hover/stat:border-purple-400/50 group-hover/stat:bg-purple-400/10 transition-all">
                            <Activity size={14} className="text-purple-400 group-hover/stat:animate-pulse" />
                         </div>
                         <div className="text-left">
                             <div className="text-[10px] text-gray-500 group-hover/stat:text-purple-400 transition-colors">LOAD</div>
                             <div className="text-xs font-mono text-white group-hover/stat:text-purple-400 transition-colors">12%</div>
                         </div>
                     </div>
                 </div>

                 {/* Floating Particles */}
                 <div className="absolute w-1 h-1 bg-brand-blue rounded-full top-1/4 left-1/4 animate-ping opacity-40 group-hover:opacity-80 group-hover:scale-150 transition-all"></div>
                 <div className="absolute w-1 h-1 bg-brand-green rounded-full bottom-1/4 right-1/4 animate-ping opacity-40 delay-500 group-hover:opacity-80 group-hover:scale-150 transition-all"></div>
                 <div className="absolute w-1.5 h-1.5 bg-white rounded-full top-1/3 right-1/3 animate-ping opacity-20 delay-1000 group-hover:opacity-60 group-hover:scale-150 transition-all"></div>
                 
                 {/* Additional animated particles on hover */}
                 <div className="absolute w-0.5 h-0.5 bg-brand-green rounded-full top-1/5 right-1/5 opacity-0 group-hover:opacity-100 group-hover:animate-ping transition-opacity" style={{ animationDelay: '0.3s' }}></div>
                 <div className="absolute w-0.5 h-0.5 bg-brand-blue rounded-full bottom-1/5 left-1/5 opacity-0 group-hover:opacity-100 group-hover:animate-ping transition-opacity" style={{ animationDelay: '0.6s' }}></div>
            </div>
            
            {/* Reflection */}
            <div className="absolute -bottom-12 left-0 w-full h-24 bg-gradient-to-t from-transparent to-[#0a0a0a] z-20"></div>
        </div>

      </main>

      {/* Stats Strip */}
      <StatsStrip />
      
      <style>{`
        @keyframes scan {
            0% { top: 0%; opacity: 0; }
            10% { opacity: 0.5; }
            90% { opacity: 0.5; }
            100% { top: 100%; opacity: 0; }
        }
        @keyframes float {
            0%, 100% { transform: translateY(0px) translateX(0px); }
            25% { transform: translateY(-20px) translateX(10px); }
            50% { transform: translateY(-40px) translateX(-10px); }
            75% { transform: translateY(-20px) translateX(5px); }
        }
        @keyframes spin-slow {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
        }
        @keyframes fadeInUp {
            from {
                opacity: 0;
                transform: translateY(10px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }
        .animate-spin-slow {
            animation: spin-slow 3s linear infinite;
        }
        .animate-pulse-slow {
            animation: pulse 4s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }
      `}</style>
    </div>
  );
};

// Stats Strip Component with animations
const StatsStrip: React.FC = () => {
  const { ref, isVisible } = useScrollAnimation(0.3);
  const stats = [
    { 
      icon: <Network className="text-brand-blue w-6 h-6" />, 
      value: 1000, 
      suffix: '+', 
      label: 'Active Nodes',
      sublabel: 'Global Network',
      color: 'brand-blue',
      bg: 'bg-brand-blue/10',
      hoverBg: 'group-hover:bg-brand-blue/20',
      hoverText: 'group-hover:text-brand-blue'
    },
    { 
      icon: <Activity className="text-brand-green w-6 h-6" />, 
      value: 99.9, 
      suffix: '%', 
      decimals: 1,
      label: 'Uptime',
      sublabel: 'Reliability',
      color: 'brand-green',
      bg: 'bg-brand-green/10',
      hoverBg: 'group-hover:bg-brand-green/20',
      hoverText: 'group-hover:text-brand-green'
    },
    { 
      icon: <Shield className="text-purple-500 w-6 h-6" />, 
      value: 256, 
      suffix: '-bit', 
      label: 'Encryption',
      sublabel: 'AES Security',
      color: 'purple-500',
      bg: 'bg-purple-500/10',
      hoverBg: 'group-hover:bg-purple-500/20',
      hoverText: 'group-hover:text-purple-400'
    },
  ];

  return (
    <div ref={ref} className="border-y border-white/5 bg-black/20 backdrop-blur-sm">
      <div className="max-w-7xl mx-auto px-4 py-8 grid grid-cols-1 md:grid-cols-3 gap-8">
        {stats.map((stat, index) => (
          <div 
            key={index}
            className={`flex items-center gap-4 justify-center group cursor-default transition-all duration-1000 ${
              isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'
            }`}
            style={{ transitionDelay: `${index * 0.15}s` }}
          >
            <div className={`p-3 ${stat.bg} rounded-lg ${stat.hoverBg} transition-all duration-300 group-hover:scale-110 group-hover:rotate-3`}>
              {stat.icon}
            </div>
            <div className="text-left">
              <div className={`text-2xl font-bold font-mono text-white ${stat.hoverText} transition-colors`}>
                <AnimatedCounter 
                  value={stat.value} 
                  suffix={stat.suffix}
                  decimals={stat.decimals}
                  isVisible={isVisible}
                />
              </div>
              <div className="text-sm font-semibold text-gray-300 mt-1">{stat.label}</div>
              <div className="text-xs text-gray-500 uppercase tracking-wider">{stat.sublabel}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default Hero;
