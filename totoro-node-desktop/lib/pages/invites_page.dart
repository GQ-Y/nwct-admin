import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../state/app_controller.dart';
import '../theme/harmony_theme.dart';
import '../widgets/harmony_widgets.dart';

class InvitesPage extends StatefulWidget {
  const InvitesPage({super.key, required this.controller});

  final AppController controller;

  @override
  State<InvitesPage> createState() => _InvitesPageState();
}

class _InvitesPageState extends State<InvitesPage> {
  late final TextEditingController ttl = TextEditingController(
    text: widget.controller.inviteTtlSeconds.toString(),
  );
  late final TextEditingController maxUses = TextEditingController(
    text: widget.controller.inviteMaxUses.toString(),
  );
  final TextEditingController revokeId = TextEditingController();

  @override
  void dispose() {
    ttl.dispose();
    maxUses.dispose();
    revokeId.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = widget.controller;
    final last = c.lastInvite;
    return ListView(
      children: [
        HarmonyCard(
          glass: true,
          title: const Text('生成邀请码'),
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
              Row(
                children: [
                  Expanded(
                    child: HarmonyField(
                      controller: ttl,
                      label: 'TTL(s)',
                      keyboardType: TextInputType.number,
                      hintText: '例如 86400',
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: HarmonyField(
                      controller: maxUses,
                      label: '次数',
                      keyboardType: TextInputType.number,
                      hintText: '例如 50',
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              HarmonyButton(
                onPressed: c.loading
                    ? null
                    : () async {
                        final t = int.tryParse(ttl.text.trim()) ?? 0;
                        final m = int.tryParse(maxUses.text.trim()) ?? 0;
                        c.inviteTtlSeconds = t;
                        c.inviteMaxUses = m;
                        await c.persist();
                        await c.createInvite();
                        if (!context.mounted) return;
                        showToast(context, '已生成');
                      },
                child: const Text('生成邀请码'),
              ),
              const SizedBox(height: 10),
              const Text(
                '提示：生成邀请码需要填写 X-Node-Key。',
                style: TextStyle(
                  color: HarmonyColors.textSecondary,
                  fontSize: 12,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        HarmonyCard(
          title: const Text('最近一次生成结果'),
          child: last == null
              ? const Text(
                  '暂无',
                  style: TextStyle(color: HarmonyColors.textSecondary),
                )
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _line(
                      'invite_id',
                      last.inviteId,
                      onCopy: () async {
                        await Clipboard.setData(
                          ClipboardData(text: last.inviteId),
                        );
                        if (!context.mounted) return;
                        showToast(context, '已复制 invite_id');
                      },
                    ),
                    const SizedBox(height: 10),
                    _line(
                      'code',
                      last.code,
                      onCopy: () async {
                        await Clipboard.setData(ClipboardData(text: last.code));
                        if (!context.mounted) return;
                        showToast(context, '已复制 code');
                      },
                    ),
                    const SizedBox(height: 10),
                    _line('expires_at', last.expiresAt),
                    const SizedBox(height: 14),
                    Row(
                      children: [
                        Expanded(
                          child: HarmonyButton(
                            variant: HarmonyButtonVariant.outline,
                            onPressed: c.loading
                                ? null
                                : () async {
                                    await c.revokeInvite(last.inviteId);
                                    if (!context.mounted) return;
                                    showToast(context, '已撤销');
                                  },
                            child: const Text('撤销此邀请码'),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
        ),
        const SizedBox(height: 16),
        HarmonyCard(
          title: const Text('撤销邀请码（删除）'),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              HarmonyField(
                controller: revokeId,
                label: 'invite_id',
                hintText: '例如 inv_...',
              ),
              const SizedBox(height: 14),
              HarmonyButton(
                variant: HarmonyButtonVariant.outline,
                onPressed: c.loading
                    ? null
                    : () async {
                        await c.revokeInvite(revokeId.text);
                        if (!context.mounted) return;
                        showToast(context, '已撤销');
                      },
                child: const Text('撤销'),
              ),
              const SizedBox(height: 8),
              const Text(
                '说明：节点侧只提供“撤销 invite_id”接口；暂无列表接口。',
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

  Widget _line(String k, String v, {Future<void> Function()? onCopy}) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 110,
          child: Text(
            k,
            style: const TextStyle(
              color: HarmonyColors.textSecondary,
              fontSize: 12,
            ),
          ),
        ),
        Expanded(
          child: Text(v, style: const TextStyle(fontWeight: FontWeight.w700)),
        ),
        if (onCopy != null) ...[
          const SizedBox(width: 12),
          HarmonyButton(
            variant: HarmonyButtonVariant.ghost,
            onPressed: () => onCopy(),
            child: const Text('复制'),
          ),
        ],
      ],
    );
  }
}
