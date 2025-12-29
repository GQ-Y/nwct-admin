
import React from 'react';
import { Language } from '../types';
import { TRANSLATIONS } from '../constants';
import { AppWindow, Apple, Terminal, HardDrive, Server, LayoutTemplate } from 'lucide-react';
import { useScrollAnimation } from '../hooks/useScrollAnimation';

interface SoftwareSectionProps {
  lang: Language;
}

const SoftwareSection: React.FC<SoftwareSectionProps> = ({ lang }) => {
  const t = TRANSLATIONS[lang].software;
  const { ref, isVisible } = useScrollAnimation(0.2);

  return (
    <section ref={ref} className="bg-[#080808] border-t border-white/5 py-24 relative overflow-hidden">
      {/* Decorative background elements */}
      <div className="absolute top-0 right-0 w-1/3 h-full bg-gradient-to-l from-brand-blue/5 to-transparent pointer-events-none"></div>
      
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="flex flex-col md:flex-row items-center justify-between gap-16">
          
          <div className={`flex-1 space-y-8 transition-all duration-1000 ${
            isVisible ? 'opacity-100 translate-x-0' : 'opacity-0 -translate-x-8'
          }`}>
            <h2 className="text-3xl md:text-5xl font-bold text-white leading-tight">
              {t.title}
            </h2>
            <p className="text-gray-400 text-lg leading-relaxed">
              {t.subtitle}
            </p>
            
            <div className="flex flex-wrap gap-4 mt-8">
               <div className="flex items-center gap-3 bg-[#121212] border border-white/10 px-5 py-3 rounded-lg shadow-lg">
                  <div className="p-2 bg-brand-green/10 rounded-lg">
                    <LayoutTemplate className="text-brand-green w-6 h-6" />
                  </div>
                  <div>
                    <div className="text-white font-bold text-sm">S1 Pro Features</div>
                    <div className="text-xs text-gray-500">{t.feature_parity}</div>
                  </div>
               </div>
            </div>
          </div>

          <div className={`flex-1 w-full transition-all duration-1000 ${
            isVisible ? 'opacity-100 translate-x-0' : 'opacity-0 translate-x-8'
          }`} style={{ transitionDelay: '0.2s' }}>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
               <PlatformCard icon={<AppWindow size={28} />} name="Windows" desc="10 / 11 / Server" delay={0} isVisible={isVisible} />
               <PlatformCard icon={<Apple size={28} />} name="macOS" desc="Intel & Silicon" delay={0.1} isVisible={isVisible} />
               <PlatformCard icon={<Terminal size={28} />} name="Linux" desc="Ubuntu / CentOS" delay={0.2} isVisible={isVisible} />
               <PlatformCard icon={<Server size={28} />} name={t.platforms.synology} desc="DSM 6.0+" highlight delay={0.3} isVisible={isVisible} />
               <PlatformCard icon={<HardDrive size={28} />} name={t.platforms.fnos} desc="Native Support" highlight delay={0.4} isVisible={isVisible} />
               <PlatformCard icon={<LayoutTemplate size={28} />} name="Docker" desc="Container" delay={0.5} isVisible={isVisible} />
            </div>
          </div>

        </div>
      </div>
    </section>
  );
};

const PlatformCard = ({ icon, name, desc, highlight, delay = 0, isVisible }: { icon: React.ReactNode, name: string, desc: string, highlight?: boolean, delay?: number, isVisible?: boolean }) => (
  <div 
    className={`p-6 rounded-xl border flex flex-col items-center text-center gap-3 transition-all duration-500 hover:-translate-y-2 hover:scale-105 ${
      highlight ? 'bg-brand-blue/5 border-brand-blue/20 shadow-[0_0_15px_rgba(33,150,243,0.1)] hover:shadow-[0_0_25px_rgba(33,150,243,0.2)]' : 'bg-[#121212] border-white/5 hover:border-white/10'
    } ${
      isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'
    }`}
    style={{ transitionDelay: `${delay}s` }}
  >
     <div className={`transition-transform duration-300 hover:scale-110 ${highlight ? 'text-brand-blue' : 'text-gray-300'}`}>{icon}</div>
     <div>
        <div className={`font-bold text-sm md:text-base ${highlight ? 'text-white' : 'text-gray-300'}`}>{name}</div>
        <div className="text-[10px] text-gray-500 uppercase tracking-wider mt-1">{desc}</div>
     </div>
  </div>
);

export default SoftwareSection;
