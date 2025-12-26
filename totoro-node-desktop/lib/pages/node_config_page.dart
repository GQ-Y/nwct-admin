import 'package:flutter/material.dart';

import '../state/app_controller.dart';
import '../theme/harmony_theme.dart';
import '../widgets/harmony_widgets.dart';

class NodeConfigPage extends StatefulWidget {
  const NodeConfigPage({super.key, required this.controller});

  final AppController controller;

  @override
  State<NodeConfigPage> createState() => _NodeConfigPageState();
}

class _NodeConfigPageState extends State<NodeConfigPage> {
  bool public = false;
  bool httpEnabled = false;
  bool httpsEnabled = false;

  final TextEditingController bridgeUrl = TextEditingController();
  final TextEditingController domainSuffix = TextEditingController();
  final TextEditingController description = TextEditingController();

  @override
  void initState() {
    super.initState();
    _syncFromDraft();
    widget.controller.addListener(_syncFromController);
    WidgetsBinding.instance.addPostFrameCallback(
      (_) => widget.controller.refreshConfig(),
    );
  }

  @override
  void dispose() {
    widget.controller.removeListener(_syncFromController);
    // 持久化当前表单草稿（离线也能继续编辑）
    widget.controller.updateDraft(
      public: public,
      httpEnabled: httpEnabled,
      httpsEnabled: httpsEnabled,
      bridgeUrl: bridgeUrl.text,
      domainSuffix: domainSuffix.text,
      description: description.text,
      persistNow: true,
      notify: false,
    );
    bridgeUrl.dispose();
    domainSuffix.dispose();
    description.dispose();
    super.dispose();
  }

  void _syncFromController() {
    final cfg = widget.controller.config;
    if (cfg == null) return;
    setState(() {
      public = cfg.public;
      httpEnabled = cfg.httpEnabled;
      httpsEnabled = cfg.httpsEnabled;
      bridgeUrl.text = cfg.bridgeUrl;
      domainSuffix.text = cfg.domainSuffix;
      description.text = cfg.description;
    });
  }

  void _syncFromDraft() {
    final d = widget.controller;
    setState(() {
      public = d.draftPublic;
      httpEnabled = d.draftHttpEnabled;
      httpsEnabled = d.draftHttpsEnabled;
      bridgeUrl.text = d.draftBridgeUrl;
      domainSuffix.text = d.draftDomainSuffix;
      description.text = d.draftDescription;
    });
  }

  @override
  Widget build(BuildContext context) {
    final c = widget.controller;
    return ListView(
      children: [
        HarmonyCard(
          glass: true,
          title: const Text('节点配置'),
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
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                value: public,
                onChanged: c.loading ? null : (v) => setState(() => public = v),
                title: const Text(
                  '公开节点（public）',
                  style: TextStyle(fontWeight: FontWeight.w700),
                ),
                subtitle: const Text(
                  '开启后该节点可在桥梁平台公开列表展示',
                  style: TextStyle(
                    color: HarmonyColors.textSecondary,
                    fontSize: 12,
                  ),
                ),
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(
                    child: SwitchListTile(
                      contentPadding: EdgeInsets.zero,
                      value: httpEnabled,
                      onChanged: c.loading
                          ? null
                          : (v) => setState(() => httpEnabled = v),
                      title: const Text(
                        'HTTP',
                        style: TextStyle(fontWeight: FontWeight.w700),
                      ),
                      subtitle: const Text(
                        '允许 HTTP 入口',
                        style: TextStyle(
                          color: HarmonyColors.textSecondary,
                          fontSize: 12,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: SwitchListTile(
                      contentPadding: EdgeInsets.zero,
                      value: httpsEnabled,
                      onChanged: c.loading
                          ? null
                          : (v) => setState(() => httpsEnabled = v),
                      title: const Text(
                        'HTTPS',
                        style: TextStyle(fontWeight: FontWeight.w700),
                      ),
                      subtitle: const Text(
                        '允许 HTTPS 入口',
                        style: TextStyle(
                          color: HarmonyColors.textSecondary,
                          fontSize: 12,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 14),
              HarmonyField(
                controller: domainSuffix,
                label: '根域名（domain_suffix）',
                hintText: '例如 frpc.example.com',
              ),
              const SizedBox(height: 12),
              HarmonyField(
                controller: bridgeUrl,
                label: 'Bridge URL（bridge_url）',
                hintText: '例如 http://127.0.0.1:18090',
              ),
              const SizedBox(height: 12),
              HarmonyField(
                controller: description,
                label: '描述（description）',
                hintText: '纯文本描述',
                maxLines: 2,
              ),
              const SizedBox(height: 18),
              Row(
                children: [
                  Expanded(
                    child: HarmonyButton(
                      onPressed: c.loading
                          ? null
                          : () async {
                              await c.saveConfig(
                                public: public,
                                httpEnabled: httpEnabled,
                                httpsEnabled: httpsEnabled,
                                description: description.text.trim(),
                                domainSuffix: domainSuffix.text.trim(),
                                bridgeUrl: bridgeUrl.text.trim(),
                              );
                              if (!context.mounted) return;
                              showToast(context, '已保存');
                            },
                      child: const Text('保存配置'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  HarmonyButton(
                    variant: HarmonyButtonVariant.outline,
                    onPressed: c.loading ? null : c.refreshConfig,
                    child: const Text('刷新'),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              const Text(
                '提示：保存配置需要填写 X-Node-Key。',
                style: TextStyle(
                  color: HarmonyColors.textSecondary,
                  fontSize: 12,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
