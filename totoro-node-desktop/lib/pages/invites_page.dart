import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../api/node_api.dart';
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
    text: widget.controller.inviteTtlDays.toString(),
  );
  late final TextEditingController maxUses = TextEditingController(
    text: widget.controller.inviteMaxUses.toString(),
  );

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await widget.controller.refreshInvites();
    });
  }

  @override
  void dispose() {
    ttl.dispose();
    maxUses.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = widget.controller;
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
                      label: '有效期（天）',
                      keyboardType: TextInputType.number,
                      hintText: '例如 1（填 0 表示不过期）',
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: HarmonyField(
                      controller: maxUses,
                      label: '可用次数',
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
                        c.inviteTtlDays = t;
                        c.inviteMaxUses = m;
                        await c.persist();
                        await c.createInvite();
                        if (!context.mounted) return;
                        showToast(context, '已生成');
                      },
                child: const Text('生成邀请码'),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        HarmonyCard(
          glass: true,
          title: const Text('邀请码列表'),
          extra: HarmonyButton(
            variant: HarmonyButtonVariant.ghost,
            onPressed: c.loading ? null : c.refreshInvites,
            child: const Text('刷新'),
          ),
          child: _InviteList(
            loading: c.loading,
            items: c.invites,
            onRevoke: (id) async {
              await c.revokeInvite(id);
              if (!context.mounted) return;
              showToast(context, '已删除');
            },
            onCopy: (label, text) async {
              await Clipboard.setData(ClipboardData(text: text));
              if (!context.mounted) return;
              showToast(context, '已复制 $label');
            },
          ),
        ),
      ],
    );
  }
}

class _InviteList extends StatelessWidget {
  const _InviteList({
    required this.loading,
    required this.items,
    required this.onRevoke,
    required this.onCopy,
  });

  final bool loading;
  final List<InviteItem> items;
  final Future<void> Function(String inviteId) onRevoke;
  final Future<void> Function(String label, String text) onCopy;

  @override
  Widget build(BuildContext context) {
    if (items.isEmpty) {
      return Text(
        loading ? '加载中…' : '暂无（仅显示本节点通过 Node API 创建并记录的邀请码）',
        style: const TextStyle(color: HarmonyColors.textSecondary),
      );
    }
    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: items.length,
      separatorBuilder: (_, __) => const Divider(height: 22),
      itemBuilder: (context, i) {
        final it = items[i];
        final statusText = it.revoked ? '已撤销' : '可用';
        final statusColor = it.revoked
            ? HarmonyColors.textSecondary
            : HarmonyColors.primary;
        final code = it.code.trim();
        final title = code.isNotEmpty ? code : '（无邀请码）';
        final expires = _fmtExpires(it.expiresAt);
        final usesText = it.maxUses <= 0 ? '不限' : '${it.used}/${it.maxUses}';
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    title,
                    style: const TextStyle(fontWeight: FontWeight.w800),
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 6,
                  ),
                  decoration: BoxDecoration(
                    color: statusColor.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    statusText,
                    style: TextStyle(
                      color: statusColor,
                      fontSize: 12,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Wrap(
              spacing: 10,
              runSpacing: 8,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [_kv('到期时间', expires), _kv('使用次数', usesText)],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                HarmonyButton(
                  variant: HarmonyButtonVariant.ghost,
                  onPressed: code.isEmpty
                      ? null
                      : () => onCopy(
                          '分享文案',
                          '我自己的高速Totoro节点，分享给你啦～好朋友就是要一起快乐～，邀请码：$code；',
                        ),
                  child: const Text('分享邀请码'),
                ),
                const SizedBox(width: 8),
                if (code.isNotEmpty)
                  HarmonyButton(
                    variant: HarmonyButtonVariant.ghost,
                    onPressed: () => onCopy('邀请码', code),
                    child: const Text('复制邀请码'),
                  ),
                const Spacer(),
                HarmonyButton(
                  variant: HarmonyButtonVariant.outline,
                  onPressed: (loading || it.revoked)
                      ? null
                      : () => onRevoke(it.inviteId),
                  child: const Text('删除'),
                ),
              ],
            ),
          ],
        );
      },
    );
  }

  static Widget _kv(String k, String v) {
    return Text(
      '$k: $v',
      style: const TextStyle(
        color: HarmonyColors.textSecondary,
        fontSize: 12,
        fontWeight: FontWeight.w600,
      ),
    );
  }

  static String _fmtExpires(String raw) {
    final s = raw.trim();
    if (s.isEmpty) return '不过期';
    try {
      final dt = DateTime.parse(s).toLocal();
      String two(int v) => v.toString().padLeft(2, '0');
      return '${dt.year}-${two(dt.month)}-${two(dt.day)} ${two(dt.hour)}:${two(dt.minute)}';
    } catch (_) {
      return s;
    }
  }
}
