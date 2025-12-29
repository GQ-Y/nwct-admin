
import React, { useState } from 'react';
import Navbar from './components/Navbar';
import Hero from './components/Hero';
import ComparisonTable from './components/ComparisonTable';
import Footer from './components/Footer';
import ProductDetail from './components/ProductDetail';
import Nodes from './components/Nodes';
import SoftwareSection from './components/SoftwareSection';
import UseCases from './components/UseCases';
import Contributors from './components/Contributors';
import { Language, ViewState } from './types';

const App: React.FC = () => {
  const [lang, setLang] = useState<Language>('zh');
  const [currentView, setCurrentView] = useState<ViewState>('home');

  const renderContent = () => {
    if (currentView === 'home') {
      return (
        <>
          <Hero lang={lang} onNavigate={setCurrentView} />
          <SoftwareSection lang={lang} />
          <UseCases lang={lang} />
          <ComparisonTable lang={lang} />
        </>
      );
    }
    
    if (currentView === 'nodes') {
      return <Nodes lang={lang} />;
    }

    // Product Views
    return (
      <>
        <ProductDetail lang={lang} type={currentView} />
        <ComparisonTable lang={lang} />
      </>
    );
  };

  return (
    <div className="min-h-screen bg-brand-dark text-white font-sans selection:bg-brand-green selection:text-black">
      <Navbar lang={lang} setLang={setLang} onNavigate={setCurrentView} currentView={currentView} />
      {renderContent()}
      <Contributors lang={lang} />
      <Footer lang={lang} />
    </div>
  );
};

export default App;
