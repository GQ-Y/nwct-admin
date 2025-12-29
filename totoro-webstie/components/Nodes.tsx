import React, { useState } from 'react';
import { Language } from '../types';
import { TRANSLATIONS } from '../constants';
import { Download, Terminal, AppWindow, Apple, Server, Users, Lock, Trophy, ShoppingCart, Globe, Copy, Check, Cpu, Layers } from 'lucide-react';

interface NodesProps {
  lang: Language;
}

const Nodes: React.FC<NodesProps> = ({ lang }) => {
  const t = TRANSLATIONS[lang].nodes;
  const [os, setOs] = useState<'linux' | 'mac' | 'windows'>('linux');
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText('curl -fsSL https://get.totoro.link/node.sh | bash -s -- --bridge=YOUR_BRIDGE_URL');
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const renderDownloadContent = () => {
    switch (os) {
        case 'linux':
            return (
                <div className="flex flex-col items-center">
                    <div className="bg-black border border-white/10 rounded-xl p-6 w-full max-w-3xl mx-auto font-mono text-sm md:text-base text-left group relative shadow-inner">
                        <div className="text-gray-500 mb-3 select-none flex items-center justify-between gap-3">
                            <span className="flex items-center gap-2">
                                <span># Install via CLI</span>
                            </span>
                            <div className="flex items-center gap-2">
                                <span className="text-xs border border-brand-green/30 bg-brand-green/10 text-brand-green rounded-full px-2.5 py-0.5 font-medium">Recommended</span>
                                <button 
                                    onClick={handleCopy}
                                    className="text-gray-500 hover:text-white transition-colors p-1.5 hover:bg-white/5 rounded-md"
                                    title={copied ? 'Copied!' : 'Copy command'}
                                >
                                    {copied ? <Check size={16} className="text-brand-green"/> : <Copy size={16} />}
                                </button>
                            </div>
                        </div>
                        <code className="text-brand-green break-all block pr-2">curl -fsSL https://get.totoro.link/node.sh | bash</code>
                    </div>
                    
                    <div className="mt-6 flex flex-wrap gap-2 justify-center">
                        <span className="px-3 py-1 rounded-full bg-white/5 border border-white/10 text-xs text-gray-400 font-mono">amd64</span>
                        <span className="px-3 py-1 rounded-full bg-white/5 border border-white/10 text-xs text-gray-400 font-mono">arm64</span>
                        <span className="px-3 py-1 rounded-full bg-white/5 border border-white/10 text-xs text-gray-400 font-mono">armv7</span>
                        <span className="px-3 py-1 rounded-full bg-white/5 border border-white/10 text-xs text-brand-green/80 border-brand-green/20 font-mono">riscv64 (Luckfox)</span>
                    </div>
                    <p className="mt-3 text-gray-500 text-sm">{t.linux_arch}</p>
                </div>
            );
        case 'mac':
            return (
                <div className="flex flex-col items-center">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 w-full max-w-2xl">
                        <button className="bg-white text-black px-6 py-4 rounded-xl font-bold flex flex-col items-center gap-2 hover:bg-gray-200 transition-colors border border-transparent">
                            <Apple size={32} />
                            <span>{t.mac_chips.silicon}</span>
                            <span className="text-xs font-normal text-gray-600 bg-gray-200 px-2 py-0.5 rounded-full">.pkg installer</span>
                        </button>
                        <button className="bg-[#1a1a1a] text-white px-6 py-4 rounded-xl font-bold flex flex-col items-center gap-2 hover:bg-[#252525] transition-colors border border-white/10">
                            <Cpu size={32} />
                            <span>{t.mac_chips.intel}</span>
                            <span className="text-xs font-normal text-gray-500 bg-white/5 px-2 py-0.5 rounded-full">.pkg installer</span>
                        </button>
                    </div>
                    <a href="#" className="mt-6 text-gray-500 hover:text-brand-green text-sm underline underline-offset-4">{t.guide}</a>
                </div>
            );
        case 'windows':
            return (
                <div className="flex flex-col items-center gap-4">
                     <button className="bg-blue-600 text-white px-8 py-4 rounded-xl font-bold flex items-center gap-3 hover:bg-blue-500 transition-colors shadow-lg shadow-blue-900/20">
                        <AppWindow size={24} />
                        <div className="text-left">
                            <div className="leading-none mb-1">Download for Windows</div>
                            <div className="text-xs text-blue-200 font-mono font-normal">x64 .msi Installer</div>
                        </div>
                    </button>
                    <div className="text-sm text-gray-500 flex items-center gap-2">
                        <span>Requires Windows 10/11 or Server 2016+</span>
                    </div>
                     <a href="#" className="text-gray-500 hover:text-brand-green text-sm underline underline-offset-4">{t.guide}</a>
                </div>
            );
    }
  }

  return (
    <div className="min-h-screen bg-[#0a0a0a] pt-16">
      {/* Header */}
      <section className="py-20 px-4 relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,#111_10%,#000_100%)]"></div>
        <div className="absolute top-0 left-0 w-full h-full opacity-10" style={{ backgroundImage: 'radial-gradient(#00C853 1px, transparent 1px)', backgroundSize: '40px 40px' }}></div>
        
        <div className="relative z-10 max-w-4xl mx-auto text-center">
            <h1 className="text-4xl md:text-6xl font-bold text-white mb-6 tracking-tight">{t.title}</h1>
            <p className="text-gray-400 text-lg md:text-xl max-w-2xl mx-auto">{t.subtitle}</p>
        </div>
      </section>

      {/* OS Download Center */}
      <section className="max-w-6xl mx-auto px-6 -mt-10 relative z-20 mb-12">
        <div className="bg-[#121212] border border-white/10 rounded-2xl p-2 md:p-8 shadow-2xl">
            <div className="flex justify-center gap-2 md:gap-8 border-b border-white/5 pb-6 mb-8">
                <button 
                    onClick={() => setOs('linux')}
                    className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-all ${os === 'linux' ? 'bg-white/10 text-brand-green' : 'text-gray-400 hover:text-white'}`}
                >
                    <Terminal size={20} />
                    <span className="font-mono">{t.os.linux}</span>
                </button>
                <button 
                     onClick={() => setOs('windows')}
                    className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-all ${os === 'windows' ? 'bg-white/10 text-blue-400' : 'text-gray-400 hover:text-white'}`}
                >
                    <AppWindow size={20} />
                    <span className="font-mono">{t.os.windows}</span>
                </button>
                <button 
                     onClick={() => setOs('mac')}
                    className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-all ${os === 'mac' ? 'bg-white/10 text-gray-200' : 'text-gray-400 hover:text-white'}`}
                >
                    <Apple size={20} />
                    <span className="font-mono">{t.os.mac}</span>
                </button>
            </div>

            <div className="py-4 min-h-[200px] flex flex-col justify-center">
                {renderDownloadContent()}
            </div>
        </div>
      </section>

      {/* Supported Platforms Grid */}
      <section className="max-w-6xl mx-auto px-6 mb-24">
          <div className="border-t border-white/5 pt-12">
              <h3 className="text-center text-gray-400 text-sm font-mono uppercase tracking-widest mb-8">{t.platforms.title}</h3>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
                  <div className="bg-white/5 rounded-lg p-4 border border-white/5 flex flex-col items-center justify-center gap-2">
                        <Layers className="text-brand-green w-6 h-6" />
                        <span className="text-white font-bold">Linux</span>
                        <span className="text-xs text-gray-500">Ubuntu, Debian, CentOS, Alpine</span>
                  </div>
                  <div className="bg-white/5 rounded-lg p-4 border border-white/5 flex flex-col items-center justify-center gap-2">
                        <Cpu className="text-blue-400 w-6 h-6" />
                        <span className="text-white font-bold">OpenWrt</span>
                        <span className="text-xs text-gray-500">MIPS / ARM / x86</span>
                  </div>
                   <div className="bg-white/5 rounded-lg p-4 border border-white/5 flex flex-col items-center justify-center gap-2">
                        <AppWindow className="text-blue-600 w-6 h-6" />
                        <span className="text-white font-bold">Windows</span>
                        <span className="text-xs text-gray-500">10, 11, Server 2016+</span>
                  </div>
                   <div className="bg-white/5 rounded-lg p-4 border border-white/5 flex flex-col items-center justify-center gap-2">
                        <Apple className="text-gray-200 w-6 h-6" />
                        <span className="text-white font-bold">macOS</span>
                        <span className="text-xs text-gray-500">Monterey (12) +</span>
                  </div>
              </div>
          </div>
      </section>

      {/* Deployment Modes */}
      <section className="py-12 bg-[#050505] border-t border-white/5">
        <div className="max-w-6xl mx-auto px-6">
            <div className="text-center mb-16">
                <h2 className="text-2xl md:text-3xl font-bold text-white">{t.modes.title}</h2>
            </div>
            
            <div className="grid md:grid-cols-2 gap-8">
                {/* Team Mode */}
                <div className="bg-[#0f0f0f] p-8 rounded-2xl border border-white/5 hover:border-brand-green/30 transition-all group">
                    <div className="w-12 h-12 bg-white/5 rounded-lg flex items-center justify-center mb-6 group-hover:bg-brand-green/10">
                        <Lock className="text-brand-green" />
                    </div>
                    <h3 className="text-xl font-bold text-white mb-2">{t.modes.team.title}</h3>
                    <p className="text-gray-400 leading-relaxed">{t.modes.team.desc}</p>
                </div>

                {/* Public Mode */}
                <div className="bg-[#0f0f0f] p-8 rounded-2xl border border-white/5 hover:border-brand-blue/30 transition-all group">
                    <div className="w-12 h-12 bg-white/5 rounded-lg flex items-center justify-center mb-6 group-hover:bg-brand-blue/10">
                        <Users className="text-brand-blue" />
                    </div>
                    <h3 className="text-xl font-bold text-white mb-2">{t.modes.public.title}</h3>
                    <p className="text-gray-400 leading-relaxed">{t.modes.public.desc}</p>
                </div>
            </div>
        </div>
      </section>

      {/* Contributor Program - Bento Grid */}
      <section className="py-24 bg-[#050505] text-white relative overflow-hidden">
        {/* World Map Simulation Dots */}
        <div className="absolute inset-0 opacity-20 pointer-events-none" style={{ backgroundImage: 'radial-gradient(#333 1.5px, transparent 1.5px)', backgroundSize: '20px 20px' }}></div>
        
        <div className="max-w-6xl mx-auto px-6 relative z-10">
            <div className="text-center mb-16">
            <h2 className="text-4xl font-bold bg-gradient-to-r from-green-400 to-blue-500 bg-clip-text text-transparent mb-2">
                {t.contributor.title}
            </h2>
            <p className="text-xl font-medium text-white mb-4">{t.contributor.slogan}</p>
            <p className="text-gray-400 max-w-xl mx-auto">
                {t.contributor.desc}
            </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Tech Perks */}
            <div className="p-8 rounded-3xl bg-[#121212] border border-white/10 hover:border-green-500/50 transition-all group shadow-lg">
                <div className="mb-6 inline-block p-3 rounded-full bg-green-500/10 text-green-500 group-hover:scale-110 transition-transform">
                    <Trophy size={32} />
                </div>
                <h3 className="text-xl font-bold mb-2">{t.contributor.perks.title}</h3>
                <p className="text-gray-500 text-sm leading-relaxed">{t.contributor.perks.desc}</p>
            </div>

            {/* Hardware Perks - Highlighted */}
            <div className="p-8 rounded-3xl bg-gradient-to-br from-green-900/20 to-blue-900/20 border border-green-500/30 shadow-2xl relative overflow-hidden">
                <div className="absolute -right-10 -top-10 w-32 h-32 bg-brand-green/20 blur-3xl rounded-full"></div>
                <div className="mb-6 inline-block p-3 rounded-full bg-white/10 text-white">
                    <ShoppingCart size={32} />
                </div>
                <h3 className="text-xl font-bold mb-2 text-white">{t.contributor.hardware.title}</h3>
                <p className="text-gray-300 text-sm leading-relaxed">{t.contributor.hardware.desc}</p>
            </div>

            {/* Honor */}
            <div className="p-8 rounded-3xl bg-[#121212] border border-white/10 hover:border-blue-500/50 transition-all group shadow-lg">
                <div className="mb-6 inline-block p-3 rounded-full bg-blue-500/10 text-blue-500 group-hover:scale-110 transition-transform">
                    <Globe size={32} />
                </div>
                <h3 className="text-xl font-bold mb-2">{t.contributor.honor.title}</h3>
                <p className="text-gray-500 text-sm leading-relaxed">{t.contributor.honor.desc}</p>
            </div>
            </div>

            {/* Quick Deploy Bar */}
            <div className="mt-12 p-6 rounded-2xl bg-black border border-white/10 font-mono text-sm overflow-hidden relative group">
                <div className="flex flex-col md:flex-row md:items-center justify-between mb-4 md:mb-0 gap-4">
                    <span className="text-gray-500 flex items-center gap-2">
                        <Server size={14} />
                        {t.contributor.deploy}
                    </span>
                </div>
                <div className="flex flex-col md:flex-row items-center justify-between gap-4 mt-2">
                    <code className="text-green-400 break-all w-full md:w-auto">
                        curl -fsSL https://get.totoro.link/node.sh | bash -s -- --bridge=YOUR_BRIDGE_URL
                    </code>
                     <button 
                        onClick={handleCopy}
                        className="text-gray-400 hover:text-white flex items-center gap-2 whitespace-nowrap bg-white/5 px-3 py-1.5 rounded-md transition-colors"
                    >
                        {copied ? <Check size={14} className="text-brand-green"/> : <Copy size={14} />}
                        {copied ? 'Copied' : 'Copy'}
                    </button>
                </div>
                <div className="absolute -right-4 -bottom-8 text-8xl font-black text-white/5 select-none pointer-events-none">NODE</div>
            </div>
        </div>
      </section>
    </div>
  );
};

export default Nodes;