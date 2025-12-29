
import { Language, ProductSpec, TranslationStructure, ProductDetailContent, ProductType } from './types';

export const TRANSLATIONS: Record<Language, TranslationStructure> = {
  zh: {
    nav: {
      home: '首页',
      products: '产品中心',
      nodes: '节点服务',
      developers: '开发者',
      console: '控制台',
    },
    hero: {
      badge: '重塑连接自由',
      title: '打破内网边界',
      subtitle: 'Totoro 是专为 Luckfox 优化的开源内网穿透体系。从微型嵌入式设备到高性能服务器，让每一台内网设备都拥有公网身份。',
      ctaPrimary: '了解 S1 Ultra',
      ctaSecondary: 'GitHub 源码',
    },
    stats: {
      nodesOnline: '在线节点',
      activeTunnels: '活跃隧道',
      latency: '平均延迟',
    },
    products: {
      title: '选择你的 Totoro',
      compare: '机型对比',
    },
    software: {
        title: '软硬同源，Pro 级体验',
        subtitle: '无需购买专用硬件，Totoro 软件版为您现有的设备注入 S1 Pro 同等强悍的穿透能力。',
        feature_parity: '全功能完美复刻',
        platforms: {
            synology: '群晖 Synology',
            fnos: '飞牛 FnOS'
        }
    },
    useCases: {
        title: 'Totoro 能为你做什么',
        subtitle: '部署开源 Totoro，无限拓展应用场景',
        items: {
            dev: { title: '开发调试', desc: '将本地开发环境暴露至公网，轻松调试 Webhook 接口，向客户展示 Demo，无需部署服务器。' },
            network: { title: '跨域组网', desc: '打破物理地域限制，将异地办公室、家庭网络组成虚拟局域网，实现资源无缝互访。' },
            nas: { title: 'NAS 远控', desc: '随时随地访问家中的 NAS 存储，高速传输文件，流畅播放高清影视，告别龟速转发。' },
            ops: { title: '远程运维', desc: '一键穿透 SSH/RDP/VNC 服务，远程管理服务器与工作站，故障排查快人一步。' },
            security: { title: '安防监控', desc: '内网监控摄像头视频流公网直连，无需云端中转，隐私更安全，延迟更低。' },
            iot: { title: 'IoT 远程访问', desc: '为树莓派、工控机等边缘设备提供稳定公网入口，实现远程数据采集与指令控制。' }
        }
    },
    nodes: {
        title: '全平台节点部署',
        subtitle: '无论 Linux 服务器、Windows 办公机还是 macOS 工作站，一键接入 Totoro 网络。',
        os: { linux: 'Linux 终端', mac: 'macOS', windows: 'Windows' },
        download: '下载安装包',
        guide: '查看部署文档',
        mac_chips: {
            silicon: 'Apple Silicon (M1/M2/M3)',
            intel: 'Intel 芯片 (x64)',
        },
        linux_arch: '完美支持 amd64, arm64, armv7, riscv64 架构',
        modes: {
            title: '灵活的共享模式',
            team: { title: '团队私有云', desc: '在团队内部署私有节点，实现成员间安全、高速的内网资源共享，完全隔离公网。' },
            public: { title: '公网共享节点', desc: '将闲置带宽贡献至公共池，供社区免费使用，赚取积分与荣誉。' }
        },
        platforms: {
            title: '支持的系统与内核',
            kernel_support: '内核版本要求 4.x 及以上',
            distros: 'Ubuntu, Debian, CentOS, Alpine, OpenWrt, Arch Linux'
        },
        contributor: {
            title: '节点贡献者奖励计划',
            slogan: '贡献闲置带宽，织就全球隧道。',
            desc: '运行一个节点，让 Totoro 网络更强大，同时赢取属于你的荣誉。',
            perks: { title: '技术特权', desc: '高级节点标识。在 Bridge 管理面板中获得专属勋章，解锁更高并发隧道数。' },
            hardware: { title: '硬件折扣与内测', desc: '优先获得下一代 S 系列硬件的内测资格及专属折扣券。' },
            honor: { title: '社区荣誉', desc: '贡献者名录。你的 ID 将永久记录在官网贡献者墙和 GitHub 仓库中。' },
            deploy: '极速部署 (Linux)'
        }
    },
    comparison: {
      core: '核心处理',
      positioning: '产品定位',
      ram: '运行内存',
      interface: '交互方式',
      tag: '编译标签',
      storage: '存储介质',
      target: '适用人群'
    },
    footer: {
      rights: '© 2024 Totoro Network. Open Source Hardware.',
      start: '开始连接',
      thanks: '特别鸣谢开源贡献者',
    },
  },
  en: {
    nav: {
      home: 'Home',
      products: 'Products',
      nodes: 'Nodes',
      developers: 'Developers',
      console: 'Console',
    },
    hero: {
      badge: 'Redefine Connectivity',
      title: 'Break Intranet Boundaries',
      subtitle: 'Totoro is an open-source intranet penetration system optimized for Luckfox. Giving every device a public identity.',
      ctaPrimary: 'Meet S1 Ultra',
      ctaSecondary: 'View on GitHub',
    },
    stats: {
      nodesOnline: 'Nodes Online',
      activeTunnels: 'Active Tunnels',
      latency: 'Avg Latency',
    },
    products: {
      title: 'Choose Your Totoro',
      compare: 'Comparison',
    },
    software: {
        title: 'Software Edition: Pro Experience',
        subtitle: 'Inject S1 Pro-level penetration capabilities into your existing devices without purchasing dedicated hardware.',
        feature_parity: '1:1 Feature Parity',
        platforms: {
            synology: 'Synology NAS',
            fnos: 'FnOS NAS'
        }
    },
    useCases: {
        title: 'What Can Totoro Do?',
        subtitle: 'Deploy open source Totoro, expand infinite scenarios',
        items: {
            dev: { title: 'Dev Debugging', desc: 'Expose local dev environments to public. Debug webhooks and demo to clients easily without deploying servers.' },
            network: { title: 'Cross-Region Net', desc: 'Break physical barriers. Create a virtual LAN between offices and home for seamless resource access.' },
            nas: { title: 'NAS Remote', desc: 'Access home NAS storage from anywhere. High-speed file transfer and HD streaming without bottlenecks.' },
            ops: { title: 'Remote Ops', desc: 'One-click SSH/RDP/VNC access. Manage servers and workstations remotely for faster troubleshooting.' },
            security: { title: 'Security Monitor', desc: 'Direct public access to intranet camera streams. No cloud relay needed for better privacy and lower latency.' },
            iot: { title: 'IoT Access', desc: 'Stable public entry for Raspberry Pi and industrial PCs. Enable remote data collection and control.' }
        }
    },
    nodes: {
        title: 'Deploy Anywhere',
        subtitle: 'Whether it is a Linux server, Windows office PC, or macOS workstation, connect to the Totoro network in one click.',
        os: { linux: 'Linux Terminal', mac: 'macOS', windows: 'Windows' },
        download: 'Download Installer',
        guide: 'View Guide',
        mac_chips: {
            silicon: 'Apple Silicon (M1/M2/M3)',
            intel: 'Intel Chip (x64)',
        },
        linux_arch: 'Supports amd64, arm64, armv7, riscv64 architectures',
        modes: {
            title: 'Flexible Deployment Modes',
            team: { title: 'Team Private Cloud', desc: 'Deploy private nodes within your team for secure, high-speed resource sharing, isolated from the public.' },
            public: { title: 'Public Community Node', desc: 'Contribute idle bandwidth to the public pool for free community use and earn rewards.' }
        },
        platforms: {
            title: 'Supported Platforms & Kernels',
            kernel_support: 'Kernel 4.x and above required',
            distros: 'Ubuntu, Debian, CentOS, Alpine, OpenWrt, Arch Linux'
        },
        contributor: {
            title: 'Node Contributor Program',
            slogan: 'Contribute Bandwidth, Weave the Global Tunnel.',
            desc: 'Run a node to make the Totoro network stronger and earn your honor.',
            perks: { title: 'Technical Perks', desc: 'Premium Badge. Unlock exclusive icons and higher concurrent tunnel limits.' },
            hardware: { title: 'Hardware Early Access', desc: 'Priority for next-gen S-series beta and exclusive coupons.' },
            honor: { title: 'Community Honor', desc: 'Contributor Wall. Your ID permanently etched on our website and GitHub.' },
            deploy: 'Quick Deploy (Linux)'
        }
    },
    comparison: {
      core: 'Core',
      positioning: 'Positioning',
      ram: 'Memory (RAM)',
      interface: 'Interface',
      tag: 'Build Tag',
      storage: 'Storage',
      target: 'Target User'
    },
    footer: {
      rights: '© 2024 Totoro Network. Open Source Hardware.',
      start: 'Start Connecting',
      thanks: 'Special Thanks to Open Source Contributors',
    },
  },
  ko: {
    nav: {
      home: '홈',
      products: '제품',
      nodes: '노드 서비스',
      developers: '개발자',
      console: '콘솔',
    },
    hero: {
      badge: '연결의 자유를 재정의하다',
      title: '인트라넷의 경계를 넘다',
      subtitle: 'Totoro는 Luckfox에 최적화된 오픈 소스 인트라넷 관통 시스템입니다. 모든 장치에 공용 ID를 부여합니다.',
      ctaPrimary: 'S1 Ultra 보기',
      ctaSecondary: 'GitHub 소스',
    },
    stats: {
      nodesOnline: '온라인 노드',
      activeTunnels: '활성 터널',
      latency: '평균 지연 시간',
    },
    products: {
      title: 'Totoro 선택',
      compare: '모델 비교',
    },
    software: {
        title: '소프트웨어 에디션: Pro 경험',
        subtitle: '전용 하드웨어 구매 없이 기존 장치에 S1 Pro 수준의 관통 능력을 주입하세요.',
        feature_parity: '완벽한 기능 동기화',
        platforms: {
            synology: 'Synology NAS',
            fnos: 'FnOS NAS'
        }
    },
    useCases: {
        title: 'Totoro의 기능',
        subtitle: '오픈 소스 Totoro 배포, 무한한 시나리오 확장',
        items: {
            dev: { title: '개발 디버깅', desc: '로컬 개발 환경을 공용 웹에 노출. 서버 배포 없이 웹훅 디버깅 및 데모 시연.' },
            network: { title: '지역 간 네트워킹', desc: '물리적 장벽을 허물고 사무실과 자택 간의 가상 LAN을 구축하여 원활한 리소스 액세스 실현.' },
            nas: { title: 'NAS 원격 제어', desc: '어디서나 가정용 NAS에 접속. 병목 현상 없는 고속 파일 전송 및 HD 스트리밍.' },
            ops: { title: '원격 운영', desc: '원클릭 SSH/RDP/VNC 접속. 서버 및 워크스테이션을 원격으로 관리하여 문제 해결 가속화.' },
            security: { title: '보안 모니터링', desc: '인트라넷 카메라 스트림에 직접 공용 액세스. 클라우드 릴레이 없이 프라이버시 강화 및 지연 시간 단축.' },
            iot: { title: 'IoT 원격 액세스', desc: '라즈베리 파이 및 산업용 PC를 위한 안정적인 공용 입구. 원격 데이터 수집 및 제어 가능.' }
        }
    },
    nodes: {
        title: '어디서나 배포 가능',
        subtitle: 'Linux 서버, Windows 사무용 PC, macOS 워크스테이션 등 어디서나 클릭 한 번으로 Totoro 네트워크에 연결하세요.',
        os: { linux: 'Linux 터미널', mac: 'macOS', windows: 'Windows' },
        download: '설치 프로그램 다운로드',
        guide: '가이드 보기',
        mac_chips: {
            silicon: 'Apple Silicon (M1/M2/M3)',
            intel: 'Intel Chip (x64)',
        },
        linux_arch: 'amd64, arm64, armv7, riscv64 아키텍처 지원',
        modes: {
            title: '유연한 배포 모드',
            team: { title: '팀 프라이빗 클라우드', desc: '팀 내에 비공개 노드를 배포하여 안전하고 빠른 리소스 공유를 실현하세요.' },
            public: { title: '공공 커뮤니티 노드', desc: '유휴 대역폭을 공용 풀에 기여하여 커뮤니티가 무료로 사용하게 하고 보상을 받으세요.' }
        },
        platforms: {
            title: '지원되는 플랫폼 및 커널',
            kernel_support: '커널 4.x 이상 필요',
            distros: 'Ubuntu, Debian, CentOS, Alpine, OpenWrt, Arch Linux'
        },
        contributor: {
            title: '노드 기여자 프로그램',
            slogan: '유휴 대역폭 기부로 전 세계를 잇는 터널을 만드세요.',
            desc: '노드를 운영하여 Totoro 네트워크를 강화하고 명예를 얻으세요.',
            perks: { title: '기술적 특전', desc: '프리미엄 뱃지. 브릿지 패널 전용 아이콘 및 더 높은 터널 제한 해제.' },
            hardware: { title: '하드웨어 얼리 액세스', desc: '차세대 S 시리즈 베타 테스트 참여 및 전용 할인 쿠폰.' },
            honor: { title: '커뮤니티 명예', desc: '기여자 명예의 전당. 홈페이지와 GitHub에 기여자로 영구 기록.' },
            deploy: '빠른 배포 (Linux)'
        }
    },
    comparison: {
      core: '코어',
      positioning: '포지셔닝',
      ram: '메모리',
      interface: '인터페이스',
      tag: '빌드 태그',
      storage: '저장소',
      target: '대상 사용자'
    },
    footer: {
      rights: '© 2024 Totoro Network. Open Source Hardware.',
      start: '연결 시작',
      thanks: '특별 감사: 오픈 소스 기여자',
    },
  },
  ja: {
    nav: {
      home: 'ホーム',
      products: '製品',
      nodes: 'ノード',
      developers: '開発者',
      console: 'コンソール',
    },
    hero: {
      badge: '接続の自由を再定義',
      title: 'イントラネットの境界を打破',
      subtitle: 'TotoroはLuckfox向けに最適化されたオープンソースのイントラネット貫通システムです。すべてのデバイスにパブリックIDを。',
      ctaPrimary: 'S1 Ultraを見る',
      ctaSecondary: 'GitHub ソース',
    },
    stats: {
      nodesOnline: 'オンラインノード',
      activeTunnels: 'アクティブトンネル',
      latency: '平均レイテンシ',
    },
    products: {
      title: 'Totoroを選ぶ',
      compare: 'モデル比較',
    },
    software: {
        title: 'ソフトウェア版：Proの体験を',
        subtitle: '専用ハードウェアを購入することなく、既存のデバイスにS1 Proレベルの貫通能力を注入します。',
        feature_parity: '完全な機能パリティ',
        platforms: {
            synology: 'Synology NAS',
            fnos: 'FnOS NAS'
        }
    },
    useCases: {
        title: 'Totoroができること',
        subtitle: 'オープンソースTotoroを導入し、利用シーンを無限に拡張',
        items: {
            dev: { title: '開発デバッグ', desc: 'ローカル開発環境をパブリックに公開。サーバー不要でWebhookデバッグやデモ展示が可能。' },
            network: { title: '拠点間ネットワーク', desc: '物理的な制約を打破し、オフィスと自宅間に仮想LANを構築してシームレスなリソースアクセスを実現。' },
            nas: { title: 'NASリモート操作', desc: 'どこからでも自宅のNASにアクセス。ボトルネックのない高速ファイル転送とHDストリーミング。' },
            ops: { title: 'リモート運用', desc: 'ワンクリックでSSH/RDP/VNC接続。サーバーやワークステーションを遠隔管理し、トラブルシューティングを迅速化。' },
            security: { title: 'セキュリティ監視', desc: 'イントラネットカメラの映像に直接パブリックアクセス。クラウドリレー不要でプライバシー保護と低遅延を実現。' },
            iot: { title: 'IoTリモートアクセス', desc: 'Raspberry Piや産業用PCに安定したパブリックエントリを提供。遠隔データ収集と制御を可能に。' }
        }
    },
    nodes: {
        title: '全プラットフォーム対応',
        subtitle: 'Linuxサーバー、WindowsオフィスPC、macOSワークステーション、ワンクリックでTotoroネットワークに接続。',
        os: { linux: 'Linux ターミナル', mac: 'macOS', windows: 'Windows' },
        download: 'インストーラーをダウンロード',
        guide: 'ガイドを見る',
        mac_chips: {
            silicon: 'Apple Silicon (M1/M2/M3)',
            intel: 'Intel Chip (x64)',
        },
        linux_arch: 'amd64, arm64, armv7, riscv64 アーキテクチャ対応',
        modes: {
            title: '柔軟なデプロイモード',
            team: { title: 'チームプライベートクラウド', desc: 'チーム内にプライベートノードをデプロイし、安全で高速なリソース共有を実現。' },
            public: { title: 'パブリックコミュニティノード', desc: '余剰帯域をパブリックプールに提供し、コミュニティの無料利用を支え、報酬を獲得。' }
        },
        platforms: {
            title: '対応プラットフォームとカーネル',
            kernel_support: 'カーネル4.x以上が必要',
            distros: 'Ubuntu, Debian, CentOS, Alpine, OpenWrt, Arch Linux'
        },
        contributor: {
            title: 'ノード貢献者プログラム',
            slogan: '余剰帯域を共有し、世界のトンネルを編み出そう。',
            desc: 'ノードを運営してTotoroネットワークを強化し、名誉を勝ち取りましょう。',
            perks: { title: '技術的特典', desc: 'プレミアムバッジ。専用バッジの付与と、トンネル同時接続数の拡張。' },
            hardware: { title: '先行体験と割引', desc: '次世代Sシリーズのベータ版優先購入権と特別割引。' },
            honor: { title: 'コミュニティの殿堂', desc: 'コントリビューターの殿堂。公式サイトとGitHubにあなたの名を刻みます。' },
            deploy: '高速デプロイ (Linux)'
        }
    },
    comparison: {
      core: 'コア',
      positioning: 'ポジショニング',
      ram: 'メモリ',
      interface: 'インターフェース',
      tag: 'ビルドタグ',
      storage: 'ストレージ',
      target: 'ターゲット'
    },
    footer: {
      rights: '© 2024 Totoro Network. Open Source Hardware.',
      start: '接続を開始',
      thanks: '特別謝辞：オープンソース貢献者',
    },
  },
};

export const PRODUCTS: ProductSpec[] = [
  {
    id: 'plus',
    name: 'S1 Plus',
    tagline: {
      zh: '极客起点',
      en: 'The Starter',
      ko: '더 스타터',
      ja: 'スターター',
    },
    positioning: {
      zh: '极客实验、高性价比',
      en: 'Max Value, DIY Favorite',
      ko: '최고의 가성비, DIY 추천',
      ja: '究極のコスパ、DIYに最適',
    },
    core: 'Luckfox Pico Plus',
    ram: '64MB (33MB avail)',
    interface: {
      zh: '无屏 (WebUI)',
      en: 'No Screen (WebUI)',
      ko: '스크린 없음 (WebUI)',
      ja: '画面なし (WebUI)',
    },
    tag: 'device_minimal',
    storage: {
      zh: '强制 SD 卡启动',
      en: 'SD Card Boot Only',
      ko: 'SD 카드 부팅 필수',
      ja: 'SDカード起動必須',
    },
    features: {
      zh: '精简内核、极低功耗',
      en: 'Minimal Kernel, Low Power',
      ko: '최소 커널, 저전력',
      ja: '最小カーネル、低消費電力',
    },
    target: {
      zh: '个人玩家',
      en: 'Hobbyist',
      ko: '개인 사용자',
      ja: 'ホビイスト',
    },
  },
  {
    id: 'pro',
    name: 'S1 Pro',
    tagline: {
      zh: '行业标准',
      en: 'The Standard',
      ko: '더 스탠다드',
      ja: 'スタンダード',
    },
    positioning: {
      zh: '工业监控、稳定转发',
      en: 'Industrial Stable, Full Features',
      ko: '산업급 안정성, 모든 기능 포함',
      ja: '工業級の安定性、フル機能搭載',
    },
    core: 'Luckfox Pico Pro',
    ram: '128MB',
    interface: {
      zh: '无屏 (WebUI)',
      en: 'No Screen (WebUI)',
      ko: '스크린 없음 (WebUI)',
      ja: '画面なし (WebUI)',
    },
    tag: 'device_full',
    storage: {
      zh: 'SPI Flash / SD 卡',
      en: 'SPI Flash / SD Card',
      ko: 'SPI 플래시 / SD 카드',
      ja: 'SPIフラッシュ / SDカード',
    },
    features: {
      zh: '增强型诊断工具链',
      en: 'Advanced Diagnostic Tools',
      ko: '고급 진단 도구',
      ja: '高度な診断ツール',
    },
    target: {
      zh: '中小团队',
      en: 'SMBs',
      ko: '중소기업',
      ja: '中小企業',
    },
  },
  {
    id: 'ultra',
    name: 'S1 Ultra',
    tagline: {
      zh: '旗舰智控',
      en: 'The Flagship',
      ko: '더 플래그십',
      ja: 'フラグシップ',
    },
    positioning: {
      zh: '全功能中枢、可视化管理',
      en: 'Smart Display, Flagship Control',
      ko: '스마트 디스플레이, 플래그십 제어',
      ja: 'スマート表示、フラグシップ制御',
    },
    core: 'Luckfox Pico Ultra',
    ram: '256MB+',
    interface: {
      zh: '1.47" IPS 智显屏',
      en: '1.47" IPS Smart Display',
      ko: '1.47" IPS 스마트 디스플레이',
      ja: '1.47" IPS スマート表示',
    },
    tag: 'device_display',
    storage: {
      zh: '板载 eMMC + SD 卡',
      en: 'Onboard eMMC + SD Card',
      ko: '내장 eMMC + SD 카드',
      ja: 'オンボードeMMC + SDカード',
    },
    features: {
      zh: '实时状态显示、硬件交互',
      en: 'Real-time Status, HW Interaction',
      ko: '실시간 상태, 하드웨어 상호작용',
      ja: 'リアルタイム状態、HW操作',
    },
    target: {
      zh: '资深极客',
      en: 'Power Users',
      ko: '파워 유저',
      ja: 'パワーユーザー',
    },
  },
];

export const PRODUCT_DETAILS: Record<ProductType, ProductDetailContent> = {
  plus: {
    title: { zh: '方寸之间，穿透万连', en: 'Tiny Footprint, Infinite Reach', ko: '작은 크기, 무한한 연결', ja: '極小サイズ、無限の接続' },
    desc: { 
      zh: '专为极致性价比打造的入门级神器。在 33MB 内存环境下依然稳健运行，是个人开发者的最佳起点。', 
      en: 'The entry-level beast built for extreme value. Runs robustly in 33MB RAM environments, the perfect starting point for developers.',
      ko: '최고의 가성비를 위해 제작된 엔트리 레벨. 33MB RAM 환경에서도 강력하게 실행됩니다.',
      ja: '究極のコストパフォーマンスを実現するエントリーモデル。33MB RAM環境でも堅牢に動作します。'
    },
    visualType: 'minimal',
    features: {
      highlight1: { title: { zh: '轻量化', en: 'Lightweight', ko: '경량화', ja: '軽量' }, text: '33MB RAM Optimized' },
      highlight2: { title: { zh: '零噪音', en: 'Silent', ko: '무소음', ja: '静音' }, text: 'Fanless Design' },
      highlight3: { title: { zh: '易扩展', en: 'Expandable', ko: '확장성', ja: '拡張性' }, text: 'MicroSD Boot' }
    },
    specs: {
      kernel: 'Linux 5.10 (Minimal)',
      agent: 'Totoro Go Lite v1.2',
      security: 'AES-128-GCM'
    }
  },
  pro: {
    title: { zh: '为稳定性而生', en: 'Born for Stability', ko: '안정성을 위해 탄생', ja: '安定性のために生まれた' },
    desc: { 
      zh: '工作室与中小团队的首选。内置全功能网络诊断工具链，支持 SPI Flash 启动，确保 24/7 不间断运行。', 
      en: 'The top choice for studios and SMBs. Built-in full network diagnostic toolchain, supports SPI Flash boot for 24/7 uptime.',
      ko: '스튜디오 및 중소기업을 위한 최고의 선택. 24/7 가동 시간을 보장하는 전체 네트워크 진단 도구 포함.',
      ja: 'スタジオや中小企業に最適。完全なネットワーク診断ツールチェーンを内蔵し、24時間365日の稼働を保証。'
    },
    visualType: 'led',
    features: {
      highlight1: { title: { zh: '高可靠', en: 'Reliable', ko: '신뢰성', ja: '信頼性' }, text: 'SPI Flash Boot' },
      highlight2: { title: { zh: '全工具', en: 'Toolchain', ko: '툴체인', ja: 'ツールチェーン' }, text: 'Net-Tools Included' },
      highlight3: { title: { zh: '广连接', en: 'Connectivity', ko: '연결성', ja: '接続性' }, text: '100Mbps Ethernet' }
    },
    specs: {
      kernel: 'Linux 5.10 (Standard)',
      agent: 'Totoro Go Std v1.2',
      security: 'AES-256-GCM'
    }
  },
  ultra: {
    title: { zh: '看得见的穿透，尽在掌握', en: 'Visible Tunnels, Total Control', ko: '눈으로 확인하는 터널링, 완벽한 제어', ja: '目に見えるトンネル、自由な制御' },
    desc: { 
      zh: '可视化内网穿透中心，为极致管理而生。无需打开电脑，一眼识别 IP 地址、连接数及节点状态。', 
      en: 'Visualized Intranet Hub, built for ultimate control. Check IP, connections, and node status at a glance without a PC.',
      ko: '최고의 제어를 위해 구축된 시각화된 인트라넷 허브. PC 없이도 IP, 연결 및 노드 상태를 한눈에 확인하세요.',
      ja: '究極の制御のために構築された可視化されたイントラネットハブ。PCなしでIP、接続、ノードの状態を一目で確認できます。'
    },
    visualType: 'screen',
    features: {
      highlight1: { title: { zh: '实时监控', en: 'Monitor', ko: '모니터링', ja: '監視' }, text: '1.47" IPS LCD' },
      highlight2: { title: { zh: '高性能', en: 'High Perf', ko: '고성능', ja: '高性能' }, text: 'RV1106 Core' },
      highlight3: { title: { zh: '智交互', en: 'Interaction', ko: '상호작용', ja: 'インタラクション' }, text: 'HW Integration' }
    },
    specs: {
      kernel: 'Linux 5.10 (Luckfox)',
      agent: 'Totoro Go Pro v1.2',
      driver: 'st7789_spi',
      security: 'AES-256-GCM'
    }
  }
};
