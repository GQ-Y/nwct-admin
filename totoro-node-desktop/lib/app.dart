import 'dart:io' show Platform;
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:bitsdojo_window/bitsdojo_window.dart';

import 'pages/connection_settings_page.dart';
import 'pages/invites_page.dart';
import 'pages/node_config_page.dart';
import 'state/app_controller.dart';
import 'theme/harmony_theme.dart';
import 'widgets/harmony_widgets.dart';

class TotoroNodeDesktopApp extends StatefulWidget {
  const TotoroNodeDesktopApp({super.key});

  @override
  State<TotoroNodeDesktopApp> createState() => _TotoroNodeDesktopAppState();
}

class _TotoroNodeDesktopAppState extends State<TotoroNodeDesktopApp> {
  final AppController controller = AppController();

  @override
  void initState() {
    super.initState();
    controller.init();
  }

  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final home = HomeShell(controller: controller);
    if (Platform.isWindows) {
      // Draw a custom frame on Windows to match "one unified surface" UX.
      return MaterialApp(
        debugShowCheckedModeBanner: false,
        theme: HarmonyTheme.build(),
        home: WindowBorder(color: Colors.transparent, width: 0, child: home),
      );
    }
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      theme: HarmonyTheme.build(),
      home: home,
    );
  }
}

enum _NavKey { nodeConfig, invites, settings }

class HomeShell extends StatefulWidget {
  const HomeShell({super.key, required this.controller});

  final AppController controller;

  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> {
  _NavKey _nav = _NavKey.nodeConfig;
  bool _inputReady = false;

  @override
  void initState() {
    super.initState();
    // 避免在首帧尚未完成布局时接收鼠标/触控事件，触发 “RenderBox was not laid out”
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      setState(() => _inputReady = true);
    });
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: widget.controller,
      builder: (context, _) {
        // macOS 使用透明标题栏 + fullSizeContentView 后，内容区域会顶到左上角三色按钮。
        // 这里额外下移一点，让整体更协调且不遮挡。
        final extraTopInset = Platform.isMacOS ? 18.0 : 0.0;

        Widget page = switch (_nav) {
          _NavKey.nodeConfig => NodeConfigPage(controller: widget.controller),
          _NavKey.invites => InvitesPage(controller: widget.controller),
          _NavKey.settings => ConnectionSettingsPage(
            controller: widget.controller,
          ),
        };

        final titleBar = Platform.isWindows ? const _WindowsTitleBar() : null;

        return IgnorePointer(
          ignoring: !_inputReady,
          child: Scaffold(
            body: Column(
              children: [
                if (titleBar != null) titleBar,
                Expanded(
                  child: SafeArea(
                    child: Padding(
                      padding: EdgeInsets.fromLTRB(
                        16,
                        16 + extraTopInset,
                        16,
                        16,
                      ),
                      child: Column(
                        children: [
                          Expanded(
                            child: Row(
                              children: [
                                SizedBox(
                                  width: 270,
                                  child: LayoutBuilder(
                                    builder: (context, constraints) {
                                      return _Sidebar(
                                        height: constraints.maxHeight,
                                        nav: _nav,
                                        onNav: (k) => setState(() => _nav = k),
                                        onOpenSettings: () => setState(
                                          () => _nav = _NavKey.settings,
                                        ),
                                      );
                                    },
                                  ),
                                ),
                                const SizedBox(width: 16),
                                Expanded(child: page),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _WindowsTitleBar extends StatelessWidget {
  const _WindowsTitleBar();

  @override
  Widget build(BuildContext context) {
    final c = Theme.of(context).colorScheme;
    final bg = HarmonyColors.bgSurface.withOpacity(0.72);
    return WindowTitleBarBox(
      child: Container(
        height: 44,
        color: bg,
        child: Row(
          children: [
            Expanded(
              child: MoveWindow(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 14),
                  child: Row(
                    children: [
                      const Text(
                        'Totoro',
                        style: TextStyle(fontWeight: FontWeight.w900),
                      ),
                      const SizedBox(width: 10),
                      Text(
                        'Node',
                        style: TextStyle(
                          color: HarmonyColors.textSecondary,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
            _WindowButtons(
              colors: WindowButtonColors(
                iconNormal: HarmonyColors.textSecondary,
                mouseOver: c.primary.withOpacity(0.08),
                mouseDown: c.primary.withOpacity(0.14),
                iconMouseOver: HarmonyColors.textPrimary,
                iconMouseDown: HarmonyColors.textPrimary,
              ),
              closeColors: WindowButtonColors(
                iconNormal: HarmonyColors.textSecondary,
                mouseOver: Colors.red.withOpacity(0.18),
                mouseDown: Colors.red.withOpacity(0.26),
                iconMouseOver: HarmonyColors.textPrimary,
                iconMouseDown: HarmonyColors.textPrimary,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _WindowButtons extends StatelessWidget {
  const _WindowButtons({required this.colors, required this.closeColors});

  final WindowButtonColors colors;
  final WindowButtonColors closeColors;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        MinimizeWindowButton(colors: colors),
        MaximizeWindowButton(colors: colors),
        CloseWindowButton(colors: closeColors),
      ],
    );
  }
}

class _Sidebar extends StatelessWidget {
  const _Sidebar({
    required this.height,
    required this.nav,
    required this.onNav,
    required this.onOpenSettings,
  });

  final double height;
  final _NavKey nav;
  final ValueChanged<_NavKey> onNav;
  final VoidCallback onOpenSettings;

  @override
  Widget build(BuildContext context) {
    final canPinBottom = height.isFinite && height > 0;
    const cardPadding = EdgeInsets.all(16);
    // HarmonyCard 本身会应用 padding（上下各 16）+ 边框，内部再用 height=constraints.maxHeight
    // 会导致总高度超出，从而触发底部溢出（你截图里的 BOTTOM OVERFLOW）。
    final innerHeight = math.max(0.0, height - cardPadding.vertical - 2);

    // 无论任何时序都不要把 flex 放进无限高度约束里；只有确认拿到有限高度才吸底。
    final child = canPinBottom
        ? SizedBox(
            height: innerHeight,
            child: Column(
              mainAxisSize: MainAxisSize.max,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Padding(
                  padding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  child: Text(
                    'Totoro',
                    style: TextStyle(fontWeight: FontWeight.w900, fontSize: 20),
                  ),
                ),
                const SizedBox(height: 12),
                _NavItem(
                  active: nav == _NavKey.nodeConfig,
                  text: '节点配置',
                  onTap: () => onNav(_NavKey.nodeConfig),
                ),
                _NavItem(
                  active: nav == _NavKey.invites,
                  text: '分享节点',
                  onTap: () => onNav(_NavKey.invites),
                ),
                const Spacer(),
                Align(
                  alignment: Alignment.bottomLeft,
                  child: IconButton(
                    tooltip: '连接设置',
                    onPressed: onOpenSettings,
                    icon: const Icon(Icons.settings_rounded),
                    color: HarmonyColors.textSecondary,
                  ),
                ),
              ],
            ),
          )
        : Column(
            // 退化模式：高度不确定时 shrink-wrap，避免任何 flex
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                child: Text(
                  'Totoro',
                  style: TextStyle(fontWeight: FontWeight.w900, fontSize: 20),
                ),
              ),
              const SizedBox(height: 12),
              _NavItem(
                active: nav == _NavKey.nodeConfig,
                text: '节点配置',
                onTap: () => onNav(_NavKey.nodeConfig),
              ),
              _NavItem(
                active: nav == _NavKey.invites,
                text: '邀请码',
                onTap: () => onNav(_NavKey.invites),
              ),
              const SizedBox(height: 12),
              Align(
                alignment: Alignment.centerLeft,
                child: IconButton(
                  tooltip: '连接设置',
                  onPressed: onOpenSettings,
                  icon: const Icon(Icons.settings_rounded),
                  color: HarmonyColors.textSecondary,
                ),
              ),
            ],
          );

    return HarmonyCard(glass: true, padding: cardPadding, child: child);
  }
}

class _NavItem extends StatelessWidget {
  const _NavItem({
    required this.active,
    required this.text,
    required this.onTap,
  });

  final bool active;
  final String text;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final bg = active ? HarmonyColors.primary : Colors.transparent;
    final fg = active ? Colors.white : HarmonyColors.textSecondary;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(999),
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 180),
        curve: Curves.easeOutCubic,
        height: 56,
        padding: const EdgeInsets.symmetric(horizontal: 20),
        margin: const EdgeInsets.only(bottom: 8),
        decoration: BoxDecoration(
          color: bg,
          borderRadius: BorderRadius.circular(999),
          boxShadow: active
              ? [
                  BoxShadow(
                    color: HarmonyColors.primary.withOpacity(0.30),
                    blurRadius: 12,
                    offset: const Offset(0, 4),
                  ),
                ]
              : null,
        ),
        alignment: Alignment.centerLeft,
        child: Text(
          text,
          style: TextStyle(fontWeight: FontWeight.w700, color: fg),
        ),
      ),
    );
  }
}

class _ConnectionDialog extends StatefulWidget {
  const _ConnectionDialog({required this.controller});

  final AppController controller;

  @override
  State<_ConnectionDialog> createState() => _ConnectionDialogState();
}

class _ConnectionDialogState extends State<_ConnectionDialog> {
  late final TextEditingController baseUrl = TextEditingController(
    text: widget.controller.baseUrl,
  );
  late final TextEditingController adminKey = TextEditingController(
    text: widget.controller.adminKey,
  );
  late final TextEditingController nodeKey = TextEditingController(
    text: widget.controller.nodeKey,
  );

  @override
  void dispose() {
    baseUrl.dispose();
    adminKey.dispose();
    nodeKey.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Dialog(
      insetPadding: const EdgeInsets.all(24),
      backgroundColor: Colors.transparent,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 620),
        child: HarmonyCard(
          glass: true,
          title: const Text('连接设置'),
          extra: HarmonyButton(
            variant: HarmonyButtonVariant.ghost,
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('关闭'),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              HarmonyField(
                controller: baseUrl,
                label: 'Node API Base URL',
                hintText: '例如 http://127.0.0.1:18080',
              ),
              const SizedBox(height: 12),
              HarmonyField(controller: adminKey, label: 'X-Admin-Key（可选）'),
              const SizedBox(height: 12),
              HarmonyField(
                controller: nodeKey,
                label: 'X-Node-Key（敏感操作需要）',
                obscureText: true,
              ),
              const SizedBox(height: 18),
              Row(
                children: [
                  Expanded(
                    child: HarmonyButton(
                      onPressed: () async {
                        await widget.controller.updateConnection(
                          baseUrl: baseUrl.text,
                          adminKey: adminKey.text,
                          nodeKey: nodeKey.text,
                        );
                        if (!context.mounted) return;
                        showToast(context, '已保存');
                        Navigator.of(context).pop();
                      },
                      child: const Text('保存'),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              const Text(
                '说明：\n'
                '- 若节点启用了 TOTOTO_NODE_ADMIN_KEY，则所有接口都需要填写 X-Admin-Key。\n'
                '- 保存配置/生成/撤销邀请码需要 X-Node-Key（对应 TOTOTO_NODE_KEY）。',
                style: TextStyle(
                  color: HarmonyColors.textSecondary,
                  height: 1.4,
                  fontSize: 12,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
