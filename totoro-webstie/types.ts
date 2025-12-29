
export type Language = 'zh' | 'en' | 'ko' | 'ja';

export interface TranslationStructure {
  nav: {
    home: string;
    products: string;
    nodes: string;
    developers: string;
    console: string;
  };
  hero: {
    badge: string;
    title: string;
    subtitle: string;
    ctaPrimary: string;
    ctaSecondary: string;
  };
  stats: {
    nodesOnline: string;
    activeTunnels: string;
    latency: string;
  };
  products: {
    title: string;
    compare: string;
  };
  software: {
    title: string;
    subtitle: string;
    feature_parity: string;
    platforms: {
        synology: string;
        fnos: string;
    };
  };
  useCases: {
    title: string;
    subtitle: string;
    items: {
      dev: { title: string; desc: string };
      network: { title: string; desc: string };
      nas: { title: string; desc: string };
      ops: { title: string; desc: string };
      security: { title: string; desc: string };
      iot: { title: string; desc: string };
    };
  };
  nodes: {
    title: string;
    subtitle: string;
    os: {
      linux: string;
      mac: string;
      windows: string;
    };
    download: string;
    guide: string;
    mac_chips: {
        silicon: string;
        intel: string;
    };
    linux_arch: string;
    modes: {
      title: string;
      team: { title: string; desc: string };
      public: { title: string; desc: string };
    };
    platforms: {
        title: string;
        kernel_support: string;
        distros: string;
    };
    contributor: {
      title: string;
      slogan: string;
      desc: string;
      perks: { title: string; desc: string };
      hardware: { title: string; desc: string };
      honor: { title: string; desc: string };
      deploy: string;
    };
  };
  comparison: {
    core: string;
    positioning: string;
    ram: string;
    interface: string;
    tag: string;
    storage: string;
    target: string;
  };
  footer: {
    rights: string;
    start: string;
    thanks: string;
  };
}

export type ProductType = 'plus' | 'pro' | 'ultra';
export type ViewState = 'home' | 'nodes' | ProductType;

export interface ProductSpec {
  id: ProductType;
  name: string;
  tagline: Record<Language, string>;
  positioning: Record<Language, string>;
  core: string;
  ram: string;
  interface: Record<Language, string>;
  tag: string;
  storage: Record<Language, string>;
  features: Record<Language, string>;
  target: Record<Language, string>;
}

export interface ProductDetailContent {
  title: Record<Language, string>;
  desc: Record<Language, string>;
  visualType: 'minimal' | 'led' | 'screen';
  features: {
    highlight1: { title: Record<Language, string>; text: string };
    highlight2: { title: Record<Language, string>; text: string };
    highlight3: { title: Record<Language, string>; text: string };
  };
  specs: {
    kernel: string;
    agent: string;
    driver?: string;
    security: string;
  }
}
