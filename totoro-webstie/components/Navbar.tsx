import React, { useState, useRef, useEffect } from 'react';
import { Menu, X, Globe, ChevronDown } from 'lucide-react';
import { Language, ViewState } from '../types';
import { TRANSLATIONS } from '../constants';
import TotoroLogo from './TotoroLogo';

interface NavbarProps {
  lang: Language;
  setLang: (lang: Language) => void;
  onNavigate: (view: ViewState) => void;
  currentView?: ViewState;
}

const Navbar: React.FC<NavbarProps> = ({ lang, setLang, onNavigate, currentView }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [isLangOpen, setIsLangOpen] = useState(false);
  const langDropdownRef = useRef<HTMLDivElement>(null);
  const t = TRANSLATIONS[lang].nav;

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (langDropdownRef.current && !langDropdownRef.current.contains(event.target as Node)) {
        setIsLangOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  const changeLang = (l: Language) => {
    setLang(l);
    setIsLangOpen(false);
  };

  const NavLink = ({ label, view }: { label: string; view?: ViewState }) => {
    const isActive = currentView === view;
    return (
      <button
        onClick={() => {
          if (view) onNavigate(view);
          setIsOpen(false);
        }}
        className={`relative px-3 py-2 text-sm font-medium font-mono uppercase tracking-wider transition-all duration-300 group whitespace-nowrap ${
          isActive 
            ? 'text-brand-green' 
            : 'text-gray-300 hover:text-brand-green'
        } ${view ? '' : 'text-center w-full'}`}
      >
        <span className="relative z-10 inline-block">{label}</span>
        {/* Active indicator - only show if active */}
        {isActive && view && (
          <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-brand-green rounded-full z-20"></span>
        )}
        {/* Hover underline animation - only show if not active */}
        {view && !isActive && (
          <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-brand-green rounded-full scale-x-0 group-hover:scale-x-100 transition-transform duration-300 origin-left z-10"></span>
        )}
      </button>
    );
  };

  const languages: { code: Language; label: string }[] = [
    { code: 'zh', label: '简体中文' },
    { code: 'en', label: 'English' },
    { code: 'ko', label: '한국어' },
    { code: 'ja', label: '日本語' }
  ];

  return (
    <nav className="fixed w-full z-50 top-0 left-0 border-b border-white/5 bg-brand-dark/80 backdrop-blur-lg">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity" onClick={() => onNavigate('home')}>
            <div className="flex items-center justify-center">
              <TotoroLogo size={32} className="drop-shadow-lg" />
            </div>
            <span className="text-white font-bold text-xl tracking-tight">Totoro</span>
          </div>
          
          <div className="hidden md:block">
            <nav className="ml-10 flex items-center space-x-2">
              <NavLink label={t.home} view="home" />
              <div className="relative group">
                <button className={`relative px-3 py-2 text-sm font-medium font-mono uppercase tracking-wider flex items-center gap-1.5 transition-all duration-300 rounded-md whitespace-nowrap ${
                  currentView === 'ultra' || currentView === 'pro' || currentView === 'plus'
                    ? 'text-brand-green' 
                    : 'text-gray-300 hover:text-brand-green hover:bg-white/5'
                }`}>
                  <span className="inline-block">{t.products}</span>
                  <ChevronDown className={`w-3 h-3 transition-transform duration-300 ${
                    currentView === 'ultra' || currentView === 'pro' || currentView === 'plus' 
                      ? 'rotate-180' 
                      : 'group-hover:rotate-180'
                  }`} />
                  {/* Active indicator */}
                  {(currentView === 'ultra' || currentView === 'pro' || currentView === 'plus') && (
                    <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-brand-green rounded-full z-20"></span>
                  )}
                  {/* Hover underline - only show if not active */}
                  {!(currentView === 'ultra' || currentView === 'pro' || currentView === 'plus') && (
                    <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-brand-green rounded-full scale-x-0 group-hover:scale-x-100 transition-transform duration-300 origin-left z-10"></span>
                  )}
                </button>
                {/* Invisible bridge to maintain hover state */}
                <div className="absolute left-0 top-full w-full h-2 pointer-events-none group-hover:pointer-events-auto"></div>
                <div className="absolute left-0 top-full mt-2 w-56 bg-[#1a1a1a] border border-white/10 rounded-lg shadow-2xl py-2 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-300 z-[60] backdrop-blur-sm transform translate-y-[-4px] group-hover:translate-y-0">
                  <div className="px-2 space-y-0.5">
                    <button 
                      onClick={() => onNavigate('ultra')} 
                      className={`w-full text-left px-4 py-2.5 text-sm rounded-md transition-all duration-200 flex items-center justify-between group/item ${
                        currentView === 'ultra'
                          ? 'text-brand-green bg-brand-green/10 font-medium'
                          : 'text-gray-300 hover:bg-white/5 hover:text-brand-green'
                      }`}
                    >
                      <span>S1 Ultra</span>
                      {currentView === 'ultra' ? (
                        <span className="w-1.5 h-1.5 bg-brand-green rounded-full"></span>
                      ) : (
                        <span className="w-1.5 h-1.5 bg-transparent group-hover/item:bg-brand-green/50 rounded-full transition-colors"></span>
                      )}
                    </button>
                    <button 
                      onClick={() => onNavigate('pro')} 
                      className={`w-full text-left px-4 py-2.5 text-sm rounded-md transition-all duration-200 flex items-center justify-between group/item ${
                        currentView === 'pro'
                          ? 'text-brand-green bg-brand-green/10 font-medium'
                          : 'text-gray-300 hover:bg-white/5 hover:text-brand-green'
                      }`}
                    >
                      <span>S1 Pro</span>
                      {currentView === 'pro' ? (
                        <span className="w-1.5 h-1.5 bg-brand-green rounded-full"></span>
                      ) : (
                        <span className="w-1.5 h-1.5 bg-transparent group-hover/item:bg-brand-green/50 rounded-full transition-colors"></span>
                      )}
                    </button>
                    <button 
                      onClick={() => onNavigate('plus')} 
                      className={`w-full text-left px-4 py-2.5 text-sm rounded-md transition-all duration-200 flex items-center justify-between group/item ${
                        currentView === 'plus'
                          ? 'text-brand-green bg-brand-green/10 font-medium'
                          : 'text-gray-300 hover:bg-white/5 hover:text-brand-green'
                      }`}
                    >
                      <span>S1 Plus</span>
                      {currentView === 'plus' ? (
                        <span className="w-1.5 h-1.5 bg-brand-green rounded-full"></span>
                      ) : (
                        <span className="w-1.5 h-1.5 bg-transparent group-hover/item:bg-brand-green/50 rounded-full transition-colors"></span>
                      )}
                    </button>
                  </div>
                </div>
              </div>
              <NavLink label={t.nodes} view="nodes" />
              <NavLink label={t.developers} />
            </nav>
          </div>

          <div className="hidden md:flex items-center gap-4">
            
            {/* Language Dropdown */}
            <div className="relative" ref={langDropdownRef}>
               <button 
                onClick={() => setIsLangOpen(!isLangOpen)}
                className="text-gray-400 hover:text-white flex items-center gap-2 text-sm font-mono border border-white/10 rounded-full px-3 py-1 transition-all hover:border-brand-green/50"
              >
                <Globe className="h-4 w-4" />
                <span className="uppercase">{languages.find(l => l.code === lang)?.label}</span>
                <ChevronDown className="h-3 w-3 opacity-50" />
              </button>

              {isLangOpen && (
                <div className="absolute right-0 mt-2 w-32 bg-[#1a1a1a] border border-white/10 rounded-md shadow-xl py-1 z-50">
                  {languages.map((l) => (
                    <button
                      key={l.code}
                      onClick={() => changeLang(l.code)}
                      className={`block w-full text-left px-4 py-2 text-sm transition-colors ${lang === l.code ? 'text-brand-green bg-brand-green/10' : 'text-gray-300 hover:bg-white/5 hover:text-white'}`}
                    >
                      {l.label}
                    </button>
                  ))}
                </div>
              )}
            </div>

            <button className="bg-white/5 hover:bg-white/10 border border-white/10 text-white px-4 py-1.5 rounded-md text-sm font-medium transition-colors">
              {t.console}
            </button>
          </div>

          <div className="-mr-2 flex md:hidden">
            <button
              onClick={() => setIsOpen(!isOpen)}
              className="bg-gray-800 inline-flex items-center justify-center p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-700 focus:outline-none"
            >
              {isOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile menu */}
      {isOpen && (
        <div className="md:hidden bg-[#0f0f0f] border-b border-white/10">
          <div className="px-3 pt-3 pb-4 space-y-1 flex flex-col items-center">
            <NavLink label={t.home} view="home" />
            <div className="w-full">
              <button 
                onClick={() => setIsLangOpen(!isLangOpen)} 
                className="w-full text-center text-gray-300 hover:text-brand-green px-3 py-2.5 text-sm font-medium transition-colors flex items-center justify-center gap-2 rounded-md hover:bg-white/5"
              >
                <Globe className="w-4 h-4 opacity-70" />
                <span>{languages.find(l => l.code === lang)?.label || lang.toUpperCase()}</span>
                <ChevronDown className={`w-4 h-4 transition-transform ${isLangOpen ? 'rotate-180' : ''}`} />
              </button>
              {isLangOpen && (
                <div className="mt-1 space-y-0.5 flex flex-col items-center">
                   {languages.map(l => (
                     <button 
                       key={l.code} 
                       onClick={() => { changeLang(l.code); setIsOpen(false); }} 
                       className={`w-full text-center text-sm py-2 px-3 rounded-md transition-colors ${
                         lang === l.code 
                           ? 'text-brand-green bg-brand-green/10 font-medium' 
                           : 'text-gray-400 hover:text-white hover:bg-white/5'
                       }`}
                     >
                       {l.label}
                     </button>
                   ))}
                </div>
              )}
            </div>
            <NavLink label="S1 Ultra" view="ultra" />
            <NavLink label="S1 Pro" view="pro" />
            <NavLink label="S1 Plus" view="plus" />
            <NavLink label={t.nodes} view="nodes" />
          </div>
        </div>
      )}
    </nav>
  );
};

export default Navbar;