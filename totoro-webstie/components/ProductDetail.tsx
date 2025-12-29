import React from 'react';
import { Language, ProductType } from '../types';
import { PRODUCT_DETAILS } from '../constants';
import { Activity, Cpu, Wifi, Maximize2, Zap, Server, HardDrive } from 'lucide-react';

interface ProductDetailProps {
  lang: Language;
  type: ProductType;
}

const ProductDetail: React.FC<ProductDetailProps> = ({ lang, type }) => {
  const content = PRODUCT_DETAILS[type];
  const t = (obj: Record<Language, string>) => obj[lang] || obj['en'];

  // Helper to render visual based on type
  const renderVisual = () => {
    if (content.visualType === 'screen') {
      return (
        <div className="w-80 h-48 md:w-96 md:h-56 bg-[#1a1a1a] rounded-xl border border-white/10 shadow-2xl relative flex items-center justify-center transform group-hover:scale-105 transition-transform duration-500 ease-out">
            {/* The Screen */}
            <div className="w-[85%] h-[80%] bg-black rounded-lg relative overflow-hidden border border-white/5 shadow-inner">
                {/* Screen Glow */}
                <div className="absolute top-0 right-0 w-32 h-32 bg-blue-500/20 blur-2xl rounded-full"></div>
                
                {/* Screen Content UI */}
                <div className="p-4 font-mono h-full flex flex-col justify-between relative z-10">
                    <div className="flex justify-between items-center border-b border-white/10 pb-2">
                        <span className="text-[10px] text-brand-green flex items-center gap-1">
                            <span className="w-1.5 h-1.5 bg-brand-green rounded-full animate-pulse"></span>
                            ONLINE
                        </span>
                        <span className="text-[10px] text-gray-500">ID: T-88X2</span>
                    </div>
                    
                    <div className="text-center my-2">
                            <div className="text-[10px] text-gray-400 uppercase">Downlink</div>
                            <div className="text-2xl text-white font-bold">1.24 <span className="text-sm font-normal text-gray-500">MB/s</span></div>
                    </div>

                    {/* Graph simulation */}
                    <div className="flex items-end gap-1 h-8 justify-between opacity-50">
                        {[40, 60, 30, 80, 50, 90, 70, 40, 60, 80].map((h, i) => (
                            <div key={i} style={{height: `${h}%`}} className="w-1.5 bg-brand-blue rounded-t-sm"></div>
                        ))}
                    </div>
                </div>
            </div>
            
            {/* Status LED */}
            <div className="absolute right-4 top-1/2 -translate-y-1/2 w-1 h-8 bg-[#111] rounded-full overflow-hidden">
                <div className="w-full h-full bg-brand-green shadow-[0_0_10px_#00C853] animate-pulse"></div>
            </div>
        </div>
      );
    } 
    
    if (content.visualType === 'led') {
      return (
        <div className="w-72 h-40 md:w-80 md:h-48 bg-[#1a1a1a] rounded-lg border border-white/10 shadow-2xl relative flex flex-col items-center justify-center transform group-hover:scale-105 transition-transform duration-500 ease-out">
             {/* Port Indicators */}
             <div className="flex gap-4 mb-4">
                {[1, 2, 3].map(i => (
                    <div key={i} className="w-2 h-2 rounded-full bg-gray-800 border border-gray-700 relative">
                         {i === 1 && <div className="absolute inset-0 bg-brand-blue blur-[2px] animate-pulse"></div>}
                         {i === 1 && <div className="absolute inset-0 bg-brand-blue rounded-full"></div>}
                    </div>
                ))}
             </div>
             <div className="text-gray-600 font-mono text-xs tracking-widest uppercase">Totoro S1 Pro</div>
             <div className="mt-4 w-[80%] h-1 bg-gray-800 rounded overflow-hidden">
                 <div className="h-full bg-brand-green w-2/3 animate-[pulse_2s_infinite]"></div>
             </div>
             {/* Reflection */}
             <div className="absolute top-0 left-0 w-full h-full bg-gradient-to-tr from-white/5 to-transparent rounded-lg pointer-events-none"></div>
        </div>
      );
    }

    // Minimal / Plus
    return (
        <div className="w-56 h-32 md:w-64 md:h-36 bg-[#151515] rounded-lg border border-white/10 shadow-2xl relative flex items-center justify-center transform group-hover:scale-105 transition-transform duration-500 ease-out">
            <div className="w-16 h-20 border-2 border-dashed border-white/10 rounded flex items-center justify-center">
                <span className="text-xs text-gray-600 font-mono -rotate-90">SD CARD</span>
            </div>
            <div className="absolute top-3 right-3 w-1.5 h-1.5 bg-brand-green rounded-full shadow-[0_0_8px_#00C853]"></div>
            <div className="absolute bottom-3 left-3 font-mono text-xs text-gray-500">S1+</div>
        </div>
    );
  };

  return (
    <div className="bg-[#0a0a0a] min-h-screen pt-16">
      
      {/* Hero Section */}
      <section className="relative h-[80vh] flex flex-col items-center justify-center overflow-hidden">
        <div className="absolute w-[600px] h-[600px] bg-brand-green/10 blur-[120px] rounded-full -top-48 animate-pulse-slow"></div>

        <div className="z-10 text-center space-y-6 px-4">
            <span className={`px-3 py-1 text-xs font-mono tracking-widest border rounded-full uppercase ${type === 'ultra' ? 'border-brand-green text-brand-green' : 'border-gray-500 text-gray-400'}`}>
            {type.toUpperCase()} SERIES
            </span>
            <h1 className="text-5xl md:text-8xl font-bold tracking-tighter text-white">
            Totoro <span className={`text-transparent bg-clip-text bg-gradient-to-r ${type === 'ultra' ? 'from-brand-green to-brand-blue' : type === 'pro' ? 'from-blue-400 to-purple-500' : 'from-gray-200 to-gray-500'}`}>S1 {type === 'plus' ? 'Plus' : type === 'pro' ? 'Pro' : 'Ultra'}</span>
            </h1>
            <p className="max-w-2xl mx-auto text-gray-400 text-lg md:text-xl font-light">
             {t(content.desc)}
            </p>
        </div>

        {/* CSS-Only Hardware Simulation */}
        <div className="mt-12 relative group cursor-pointer">
            {renderVisual()}
            {/* Reflection on floor */}
            <div className="absolute -bottom-10 left-4 right-4 h-8 bg-black/50 blur-xl rounded-[100%]"></div>
        </div>
      </section>

      {/* Feature Grid */}
      <section className="py-24 bg-[#080808]">
          <div className="max-w-7xl mx-auto px-6 grid md:grid-cols-2 gap-12 items-center">
              <div className="space-y-8">
                  <h2 className="text-4xl font-bold text-white">{t(content.title)}</h2>
                  <div className="space-y-6">
                      <FeatureRow icon={<Activity />} title={t(content.features.highlight1.title)} text={content.features.highlight1.text} />
                      <FeatureRow icon={type === 'ultra' ? <Cpu /> : <Server />} title={t(content.features.highlight2.title)} text={content.features.highlight2.text} />
                      <FeatureRow icon={type === 'plus' ? <HardDrive /> : <Wifi />} title={t(content.features.highlight3.title)} text={content.features.highlight3.text} />
                  </div>
              </div>
              <div className="bg-[#121212] p-8 rounded-2xl border border-white/5">
                 <h3 className="text-white font-bold text-xl mb-4 flex items-center gap-2">
                    <Maximize2 className="text-brand-blue" size={20}/>
                    System Architecture
                 </h3>
                 <div className="font-mono text-xs text-gray-400 space-y-2">
                    <div className="flex justify-between border-b border-white/5 pb-2">
                        <span>Kernel</span> <span className="text-white">{content.specs.kernel}</span>
                    </div>
                    <div className="flex justify-between border-b border-white/5 pb-2">
                        <span>Agent</span> <span className="text-white">{content.specs.agent}</span>
                    </div>
                    {content.specs.driver && (
                        <div className="flex justify-between border-b border-white/5 pb-2">
                            <span>Driver</span> <span className="text-white">{content.specs.driver}</span>
                        </div>
                    )}
                    <div className="flex justify-between border-b border-white/5 pb-2">
                        <span>Security</span> <span className="text-white">{content.specs.security}</span>
                    </div>
                 </div>
                 
                 {type !== 'ultra' && (
                    <div className="mt-6 pt-4 border-t border-white/5">
                        <div className="flex items-center gap-2 text-yellow-500/80 text-xs">
                            <Zap size={14} />
                            <span>Low Power Consumption Mode Active</span>
                        </div>
                    </div>
                 )}
              </div>
          </div>
      </section>
    </div>
  );
};

const FeatureRow = ({ icon, title, text }: { icon: React.ReactNode, title: string, text: string }) => (
    <div className="flex items-center gap-4 p-4 rounded-lg hover:bg-white/5 transition-colors border border-transparent hover:border-white/5">
        <div className="text-brand-green">{icon}</div>
        <div>
            <h4 className="text-gray-200 font-bold">{title}</h4>
            <p className="text-gray-500 text-sm font-mono">{text}</p>
        </div>
    </div>
)

export default ProductDetail;