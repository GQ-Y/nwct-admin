
import React, { useState, useEffect } from 'react';
import { Language } from '../types';
import { TRANSLATIONS } from '../constants';
import { Github, Twitter, Disc } from 'lucide-react';

interface FooterProps {
  lang: Language;
}

const Footer: React.FC<FooterProps> = ({ lang }) => {
  const t = TRANSLATIONS[lang].footer;
  const [stats, setStats] = useState({ nodes: 48, latency: 24 });

  useEffect(() => {
    // Simulate live network status updates
    const interval = setInterval(() => {
      setStats(prev => {
        // Fluctuate nodes slightly around the base
        const nodeChange = Math.random() > 0.7 ? 1 : Math.random() < 0.3 ? -1 : 0;
        const newNodes = Math.max(40, prev.nodes + nodeChange);
        
        // Fluctuate latency
        const newLatency = 20 + Math.floor(Math.random() * 15); // 20ms - 35ms

        return {
          nodes: newNodes,
          latency: newLatency
        };
      });
    }, 2000);

    return () => clearInterval(interval);
  }, []);
  
  return (
    <footer className="bg-[#020202] border-t border-white/5 pt-12 pb-12">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row justify-between items-center gap-6">
          <div className="text-center md:text-left">
             <h3 className="text-2xl font-bold text-white mb-2">Totoro</h3>
             <p className="text-gray-500 text-sm">{t.rights}</p>
          </div>

          {/* Status Bar */}
          <div className="bg-[#121212] border border-white/10 rounded-lg px-4 py-2 flex items-center gap-4 text-xs font-mono text-gray-400 shadow-inner">
             <div className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-brand-green animate-pulse"></span>
                <span className="text-brand-green/80">SYSTEM ONLINE</span>
             </div>
             <div className="hidden sm:block text-gray-700">|</div>
             <div className="hidden sm:block tabular-nums transition-all duration-300">{stats.nodes} Nodes</div>
             <div className="hidden sm:block text-gray-700">|</div>
             <div className="tabular-nums transition-all duration-300">{stats.latency}ms Latency</div>
          </div>

          <div className="flex gap-4">
            <a href="#" className="text-gray-400 hover:text-white transition-colors"><Github className="w-5 h-5" /></a>
            <a href="#" className="text-gray-400 hover:text-blue-400 transition-colors"><Twitter className="w-5 h-5" /></a>
            <a href="#" className="text-gray-400 hover:text-purple-400 transition-colors"><Disc className="w-5 h-5" /></a>
          </div>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
