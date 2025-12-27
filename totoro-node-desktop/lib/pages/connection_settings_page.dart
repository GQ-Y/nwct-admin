import 'package:flutter/material.dart';

import '../state/app_controller.dart';
import '../widgets/harmony_widgets.dart';

class ConnectionSettingsPage extends StatefulWidget {
  const ConnectionSettingsPage({super.key, required this.controller});

  final AppController controller;

  @override
  State<ConnectionSettingsPage> createState() => _ConnectionSettingsPageState();
}

class _ConnectionSettingsPageState extends State<ConnectionSettingsPage> {
  @override
  Widget build(BuildContext context) {
    final c = widget.controller;
    return ListView(
      children: [
        HarmonyCard(
          glass: true,
          title: const Text('连接设置'),
          extra: c.loading
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : null,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                '当前桌面端默认直连本机 Node API，无需配置。',
                style: TextStyle(fontWeight: FontWeight.w800),
              ),
              const SizedBox(height: 10),
              const Text(
                '默认地址：',
                style: TextStyle(fontWeight: FontWeight.w700),
              ),
              const SizedBox(height: 6),
              const Text('http://127.0.0.1:18081'),
              const SizedBox(height: 14),
              HarmonyButton(
                variant: HarmonyButtonVariant.outline,
                onPressed: c.loading ? null : c.refreshConfig,
                child: const Text('重新拉取配置'),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
