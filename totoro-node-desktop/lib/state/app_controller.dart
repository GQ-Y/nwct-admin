import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/node_api.dart';

class AppController extends ChangeNotifier {
  // 连接参数已内置（本机 Node API: 18081），不再持久化 baseUrl/adminKey/nodeKey
  static const _kInviteTtl = 'invite_ttl_seconds'; // legacy
  static const _kInviteTtlDays = 'invite_ttl_days';
  static const _kInviteMaxUses = 'invite_max_uses';
  static const _kDraftPublic = 'draft_public';
  static const _kDraftHttp = 'draft_http_enabled';
  static const _kDraftHttps = 'draft_https_enabled';
  static const _kDraftDomainSuffix = 'draft_domain_suffix';
  static const _kDraftDescription = 'draft_description';

  String baseUrl = 'http://127.0.0.1:18081';
  String adminKey = '';
  String nodeKey = '';

  bool loading = false;
  String output = 'ready';

  NodeConfig? config;
  CreateInviteResult? lastInvite;
  List<InviteItem> invites = const [];

  int inviteTtlDays = 1;
  int inviteMaxUses = 50;

  bool draftPublic = false;
  bool draftHttpEnabled = false;
  bool draftHttpsEnabled = false;
  String draftDomainSuffix = '';
  String draftDescription = '';

  Future<void> init() async {
    final p = await SharedPreferences.getInstance();
    // 强制使用内置地址，避免旧版本残留配置导致“看起来没连上”
    baseUrl = 'http://127.0.0.1:18081';
    adminKey = '';
    nodeKey = '';
    // prefer days; fallback from legacy seconds
    inviteTtlDays = p.getInt(_kInviteTtlDays) ?? inviteTtlDays;
    final legacySeconds = p.getInt(_kInviteTtl);
    if (legacySeconds != null && (p.getInt(_kInviteTtlDays) == null)) {
      // 86400 -> 1 day, 0 -> 0 (never expires)
      if (legacySeconds <= 0) {
        inviteTtlDays = 0;
      } else {
        inviteTtlDays = (legacySeconds / 86400).round();
        if (inviteTtlDays <= 0) inviteTtlDays = 1;
      }
    }
    inviteMaxUses = p.getInt(_kInviteMaxUses) ?? inviteMaxUses;

    draftPublic = p.getBool(_kDraftPublic) ?? draftPublic;
    draftHttpEnabled = p.getBool(_kDraftHttp) ?? draftHttpEnabled;
    draftHttpsEnabled = p.getBool(_kDraftHttps) ?? draftHttpsEnabled;
    draftDomainSuffix = (p.getString(_kDraftDomainSuffix) ?? draftDomainSuffix)
        .trim();
    draftDescription = (p.getString(_kDraftDescription) ?? draftDescription)
        .trim();

    notifyListeners();
  }

  Future<void> persist() async {
    final p = await SharedPreferences.getInstance();
    await p.setInt(_kInviteTtlDays, inviteTtlDays);
    await p.setInt(_kInviteMaxUses, inviteMaxUses);
  }

  Future<void> persistDraft() async {
    final p = await SharedPreferences.getInstance();
    await p.setBool(_kDraftPublic, draftPublic);
    await p.setBool(_kDraftHttp, draftHttpEnabled);
    await p.setBool(_kDraftHttps, draftHttpsEnabled);
    await p.setString(_kDraftDomainSuffix, draftDomainSuffix.trim());
    await p.setString(_kDraftDescription, draftDescription);
  }

  Future<void> updateDraft({
    required bool public,
    required bool httpEnabled,
    required bool httpsEnabled,
    required String domainSuffix,
    required String description,
    bool persistNow = false,
    bool notify = true,
  }) async {
    draftPublic = public;
    draftHttpEnabled = httpEnabled;
    draftHttpsEnabled = httpsEnabled;
    draftDomainSuffix = domainSuffix.trim();
    draftDescription = description;
    if (persistNow) await persistDraft();
    if (notify) notifyListeners();
  }

  Future<void> updateConnection({
    required String baseUrl,
    required String adminKey,
    required String nodeKey,
  }) async {
    // no-op：连接设置已移除，避免外部修改连接参数
  }

  NodeApiClient _api() {
    final b = baseUrl.trim();
    if (b.isEmpty) throw ApiException(message: 'Base URL 不能为空');
    return NodeApiClient(baseUrl: b, adminKey: adminKey, nodeKey: nodeKey);
  }

  void _setOutput(dynamic v) {
    if (v is String) {
      output = v;
    } else {
      output = const JsonEncoder.withIndent('  ').convert(v);
    }
    notifyListeners();
  }

  Future<void> refreshConfig() async {
    loading = true;
    notifyListeners();
    try {
      final cfg = await _api().getConfig();
      config = cfg;
      // 同步到草稿（离线也能继续编辑）
      draftPublic = cfg.public;
      draftHttpEnabled = cfg.httpEnabled;
      draftHttpsEnabled = cfg.httpsEnabled;
      draftDomainSuffix = cfg.domainSuffix;
      draftDescription = cfg.description;
      await persistDraft();
      _setOutput({
        'code': 0,
        'data': {
          'public': cfg.public,
          'description': cfg.description,
          'domain_suffix': cfg.domainSuffix,
          'http_enabled': cfg.httpEnabled,
          'https_enabled': cfg.httpsEnabled,
        },
      });
    } catch (e) {
      _setOutput(_errToOut(e));
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<void> saveConfig({
    required bool public,
    required bool httpEnabled,
    required bool httpsEnabled,
    required String description,
    required String domainSuffix,
  }) async {
    loading = true;
    notifyListeners();
    try {
      await updateDraft(
        public: public,
        httpEnabled: httpEnabled,
        httpsEnabled: httpsEnabled,
        domainSuffix: domainSuffix,
        description: description,
        persistNow: true,
      );
      final out = await _api().updateConfig(
        public: public,
        httpEnabled: httpEnabled,
        httpsEnabled: httpsEnabled,
        description: description,
        domainSuffix: domainSuffix,
      );
      await refreshConfig();
      _setOutput({'code': 0, 'data': out});
    } catch (e) {
      _setOutput(_errToOut(e));
      rethrow;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<void> createInvite() async {
    loading = true;
    notifyListeners();
    try {
      await persist();
      final res = await _api().createInvite(
        ttlDays: inviteTtlDays,
        maxUses: inviteMaxUses,
        scopeJson: '{}',
      );
      lastInvite = res;
      // 刷新列表（列表由节点侧本地记录生成）
      invites = await _api().listInvites(limit: 200, includeRevoked: false);
      _setOutput({
        'code': 0,
        'data': {
          'invite_id': res.inviteId,
          'code': res.code,
          'expires_at': res.expiresAt,
        },
      });
    } catch (e) {
      _setOutput(_errToOut(e));
      rethrow;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<void> revokeInvite(String inviteId) async {
    loading = true;
    notifyListeners();
    try {
      final out = await _api().revokeInvite(inviteId: inviteId);
      invites = await _api().listInvites(limit: 200, includeRevoked: false);
      _setOutput({'code': 0, 'data': out});
    } catch (e) {
      _setOutput(_errToOut(e));
      rethrow;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<void> refreshInvites() async {
    loading = true;
    notifyListeners();
    try {
      invites = await _api().listInvites(limit: 200, includeRevoked: false);
    } catch (e) {
      _setOutput(_errToOut(e));
      rethrow;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Map<String, dynamic> _errToOut(Object e) {
    if (e is ApiException) {
      return {
        'code': e.code ?? 500,
        'message': e.message,
        'http_status': e.httpStatus,
      };
    }
    return {'code': 500, 'message': e.toString()};
  }
}
