
import React from 'react';
import { Language } from '../types';
import { TRANSLATIONS } from '../constants';
import { Code2, Globe2, HardDrive, Terminal, ShieldCheck, Cpu } from 'lucide-react';
import { useScrollAnimation } from '../hooks/useScrollAnimation';

interface UseCasesProps {
  lang: Language;
}

const UseCases: React.FC<UseCasesProps> = ({ lang }) => {
  const t = TRANSLATIONS[lang].useCases;
  const { ref, isVisible } = useScrollAnimation(0.15);

  const cases = [
    { id: 'dev', icon: <Code2 size={24} />, color: 'text-blue-400', bg: 'bg-blue-400/10', border: 'hover:border-blue-400/50' },
    { id: 'network', icon: <Globe2 size={24} />, color: 'text-purple-400', bg: 'bg-purple-400/10', border: 'hover:border-purple-400/50' },
    { id: 'nas', icon: <HardDrive size={24} />, color: 'text-amber-400', bg: 'bg-amber-400/10', border: 'hover:border-amber-400/50' },
    { id: 'ops', icon: <Terminal size={24} />, color: 'text-green-400', bg: 'bg-green-400/10', border: 'hover:border-green-400/50' },
    { id: 'security', icon: <ShieldCheck size={24} />, color: 'text-red-400', bg: 'bg-red-400/10', border: 'hover:border-red-400/50' },
    { id: 'iot', icon: <Cpu size={24} />, color: 'text-cyan-400', bg: 'bg-cyan-400/10', border: 'hover:border-cyan-400/50' },
  ];

  return (
    <section ref={ref} className="py-24 bg-[#0a0a0a] relative overflow-hidden">
        {/* Background gradient splash */}
        <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[400px] bg-brand-green/5 blur-[100px] rounded-full pointer-events-none"></div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className={`text-center mb-16 transition-all duration-1000 ${
          isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'
        }`}>
          <h2 className="text-3xl md:text-5xl font-bold text-white mb-4">{t.title}</h2>
          <p className="text-gray-400 text-lg">{t.subtitle}</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {cases.map((item, index) => {
             const data = t.items[item.id as keyof typeof t.items];
             return (
                <div 
                    key={item.id}
                    className={`group relative p-8 rounded-2xl bg-[#121212] border border-white/5 transition-all duration-500 hover:-translate-y-3 hover:scale-105 hover:shadow-2xl ${item.border} ${
                      isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-12'
                    }`}
                    style={{ transitionDelay: `${index * 0.1}s` }}
                >
                    <div className={`w-12 h-12 rounded-xl flex items-center justify-center mb-6 ${item.bg} ${item.color} transition-all duration-300 group-hover:scale-125 group-hover:rotate-6`}>
                        {item.icon}
                    </div>
                    
                    <h3 className="text-xl font-bold text-white mb-3 group-hover:text-white transition-colors">
                        {data.title}
                    </h3>
                    
                    <p className="text-gray-400 leading-relaxed text-sm group-hover:text-gray-300 transition-colors">
                        {data.desc}
                    </p>

                    {/* Hover Glow Effect */}
                    <div className={`absolute inset-0 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none bg-gradient-to-br from-white/5 to-transparent`}></div>
                    
                    {/* Animated border on hover */}
                    <div className={`absolute inset-0 rounded-2xl border-2 ${item.border.replace('hover:', '')} opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none`}></div>
                </div>
             )
          })}
        </div>
      </div>
    </section>
  );
};

export default UseCases;
