import React, { useState, useRef, useEffect } from 'react';
import { Menu, X, Globe, Terminal, ChevronDown } from 'lucide-react';
import { Language, ViewState } from '../types';
import { TRANSLATIONS } from '../constants';

interface NavbarProps {
  lang: Language;
  setLang: (lang: Language) => void;
  onNavigate: (view: ViewState) => void;
}

const Navbar: React.FC<NavbarProps> = ({ lang, setLang, onNavigate }) => {
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

  const NavLink = ({ label, view }: { label: string; view?: ViewState }) => (
    <button
      onClick={() => {
        if (view) onNavigate(view);
        setIsOpen(false);
      }}
      className="text-gray-300 hover:text-brand-green transition-colors px-3 py-2 text-sm font-medium font-mono uppercase tracking-wider"
    >
      {label}
    </button>
  );

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
          <div className="flex items-center gap-2 cursor-pointer" onClick={() => onNavigate('home')}>
            <div className="bg-brand-green/10 p-1.5 rounded-lg border border-brand-green/20">
              <Terminal className="h-6 w-6 text-brand-green" />
            </div>
            <span className="text-white font-bold text-xl tracking-tight">Totoro</span>
          </div>
          
          <div className="hidden md:block">
            <div className="ml-10 flex items-baseline space-x-4">
              <NavLink label={t.home} view="home" />
              <div className="relative group">
                <button className="text-gray-300 hover:text-brand-green px-3 py-2 text-sm font-medium font-mono uppercase tracking-wider flex items-center gap-1">
                  {t.products} <ChevronDown className="w-3 h-3" />
                </button>
                <div className="absolute left-0 mt-0 w-48 bg-[#1a1a1a] border border-white/10 rounded-md shadow-lg py-1 hidden group-hover:block z-50">
                    <button onClick={() => onNavigate('ultra')} className="block w-full text-left px-4 py-2 text-sm text-gray-300 hover:bg-white/5 hover:text-brand-green transition-colors">S1 Ultra</button>
                    <button onClick={() => onNavigate('pro')} className="block w-full text-left px-4 py-2 text-sm text-gray-300 hover:bg-white/5 hover:text-brand-green transition-colors">S1 Pro</button>
                    <button onClick={() => onNavigate('plus')} className="block w-full text-left px-4 py-2 text-sm text-gray-300 hover:bg-white/5 hover:text-brand-green transition-colors">S1 Plus</button>
                </div>
              </div>
              <NavLink label={t.nodes} view="nodes" />
              <NavLink label={t.developers} />
            </div>
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
          <div className="px-2 pt-2 pb-3 space-y-1 sm:px-3 flex flex-col">
            <NavLink label={t.home} view="home" />
            <button 
              onClick={() => setIsLangOpen(!isLangOpen)} 
              className="text-left text-gray-300 px-3 py-2 text-sm font-mono uppercase flex items-center justify-between"
            >
              <span>Language: {lang}</span>
              <ChevronDown className="w-4 h-4" />
            </button>
            {isLangOpen && (
              <div className="pl-6 space-y-1 border-l border-white/10 ml-3 mb-2">
                 {languages.map(l => (
                   <button key={l.code} onClick={() => { changeLang(l.code); setIsOpen(false); }} className="block w-full text-left text-gray-400 py-1 text-sm">{l.label}</button>
                 ))}
              </div>
            )}
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