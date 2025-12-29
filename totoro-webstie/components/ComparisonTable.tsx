import React from 'react';
import { Language } from '../types';
import { PRODUCTS, TRANSLATIONS } from '../constants';

interface ComparisonTableProps {
  lang: Language;
}

const ComparisonTable: React.FC<ComparisonTableProps> = ({ lang }) => {
  const t = TRANSLATIONS[lang].comparison;

  return (
    <div className="py-24 bg-brand-dark relative overflow-hidden">
        {/* Background Grids */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:24px_24px]"></div>
      
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="text-center mb-16">
          <h2 className="text-3xl md:text-5xl font-bold text-white mb-4">{TRANSLATIONS[lang].products.compare}</h2>
          <div className="w-24 h-1 bg-brand-green mx-auto rounded-full"></div>
        </div>

        <div className="overflow-x-auto pb-8">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr>
                <th className="p-4 bg-transparent"></th>
                {PRODUCTS.map((p) => (
                  <th key={p.id} className={`p-4 min-w-[200px] text-xl font-bold ${p.id === 'ultra' ? 'text-brand-green' : 'text-white'}`}>
                    <div className="flex flex-col gap-2">
                        <span>{p.name}</span>
                        {p.id === 'ultra' && <span className="text-xs px-2 py-0.5 border border-brand-green rounded-full self-start text-brand-green bg-brand-green/10">RECOMMENDED</span>}
                    </div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-white/10 text-gray-300 font-mono text-sm">
              <tr>
                <td className="p-4 font-sans font-semibold text-gray-400">{t.core}</td>
                {PRODUCTS.map(p => <td key={p.id} className="p-4 bg-white/[0.02]">{p.core}</td>)}
              </tr>
              <tr>
                <td className="p-4 font-sans font-semibold text-gray-400">{t.positioning}</td>
                {PRODUCTS.map(p => <td key={p.id} className="p-4 bg-white/[0.02]">{p.positioning[lang]}</td>)}
              </tr>
              <tr>
                <td className="p-4 font-sans font-semibold text-gray-400">{t.ram}</td>
                {PRODUCTS.map(p => <td key={p.id} className="p-4 bg-white/[0.02]">{p.ram}</td>)}
              </tr>
              <tr>
                <td className="p-4 font-sans font-semibold text-gray-400">{t.interface}</td>
                {PRODUCTS.map(p => (
                    <td key={p.id} className={`p-4 bg-white/[0.02] ${p.id === 'ultra' ? 'text-brand-green font-bold' : ''}`}>
                        {p.interface[lang]}
                    </td>
                ))}
              </tr>
              <tr>
                <td className="p-4 font-sans font-semibold text-gray-400">{t.tag}</td>
                {PRODUCTS.map(p => <td key={p.id} className="p-4 bg-white/[0.02]"><code className="bg-black px-1 py-0.5 rounded border border-white/10 text-blue-400">{p.tag}</code></td>)}
              </tr>
              <tr>
                <td className="p-4 font-sans font-semibold text-gray-400">{t.storage}</td>
                {PRODUCTS.map(p => <td key={p.id} className="p-4 bg-white/[0.02]">{p.storage[lang]}</td>)}
              </tr>
              <tr>
                <td className="p-4 font-sans font-semibold text-gray-400">{t.target}</td>
                {PRODUCTS.map(p => <td key={p.id} className="p-4 bg-white/[0.02]">{p.target[lang]}</td>)}
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default ComparisonTable;