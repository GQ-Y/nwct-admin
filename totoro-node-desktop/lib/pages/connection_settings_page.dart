import 'package:flutter/material.dart';

import '../state/app_controller.dart';
import '../theme/harmony_theme.dart';
import '../widgets/harmony_widgets.dart';

class ConnectionSettingsPage extends StatefulWidget {
  const ConnectionSettingsPage({super.key, required this.controller});

  final AppController controller;

  @override
  State<ConnectionSettingsPage> createState() => _ConnectionSettingsPageState();
}

class _ConnectionSettingsPageState extends State<ConnectionSettingsPage> {
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
                        await c.updateConnection(
                          baseUrl: baseUrl.text,
                          adminKey: adminKey.text,
                          nodeKey: nodeKey.text,
                        );
                        if (!context.mounted) return;
                        showToast(context, '已保存');
                      },
                      child: const Text('保存'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  HarmonyButton(
                    variant: HarmonyButtonVariant.outline,
                    onPressed: () {
                      baseUrl.text = c.baseUrl;
                      adminKey.text = c.adminKey;
                      nodeKey.text = c.nodeKey;
                      showToast(context, '已恢复当前值');
                    },
                    child: const Text('恢复'),
                  ),
                ],
              ),
            ],
          ),
        ),
      ],
    );
  }
}
