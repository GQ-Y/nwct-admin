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
    text: widget.controller.inviteTtlSeconds.toString(),
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
              showToast(context, '已撤销');
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
        loading ? '加载中…' : '暂无（仅显示本节点通过 Node API 创建并记录的邀请记录）',
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
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    it.inviteId,
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
              children: [
                _kv('expires', it.expiresAt.isEmpty ? '-' : it.expiresAt),
                _kv('max_uses', it.maxUses <= 0 ? '-' : '${it.maxUses}'),
                if (it.code.trim().isNotEmpty) _kv('code', it.code.trim()),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                HarmonyButton(
                  variant: HarmonyButtonVariant.ghost,
                  onPressed: () => onCopy('invite_id', it.inviteId),
                  child: const Text('复制ID'),
                ),
                const SizedBox(width: 8),
                if (it.code.trim().isNotEmpty)
                  HarmonyButton(
                    variant: HarmonyButtonVariant.ghost,
                    onPressed: () => onCopy('code', it.code.trim()),
                    child: const Text('复制Code'),
                  ),
                const Spacer(),
                HarmonyButton(
                  variant: HarmonyButtonVariant.outline,
                  onPressed: (loading || it.revoked)
                      ? null
                      : () => onRevoke(it.inviteId),
                  child: const Text('撤销'),
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
}
