import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/node_api.dart';

class AppController extends ChangeNotifier {
  static const _kBaseUrl = 'node_base_url';
  static const _kAdminKey = 'node_admin_key';
  static const _kNodeKey = 'node_node_key';
  static const _kInviteTtl = 'invite_ttl_seconds';
  static const _kInviteMaxUses = 'invite_max_uses';
  static const _kDraftPublic = 'draft_public';
  static const _kDraftHttp = 'draft_http_enabled';
  static const _kDraftHttps = 'draft_https_enabled';
  static const _kDraftBridgeUrl = 'draft_bridge_url';
  static const _kDraftDomainSuffix = 'draft_domain_suffix';
  static const _kDraftDescription = 'draft_description';

  String baseUrl = 'http://127.0.0.1:18080';
  String adminKey = '';
  String nodeKey = '';

  bool loading = false;
  String output = 'ready';

  NodeConfig? config;
  CreateInviteResult? lastInvite;

  int inviteTtlSeconds = 86400;
  int inviteMaxUses = 50;

  bool draftPublic = false;
  bool draftHttpEnabled = false;
  bool draftHttpsEnabled = false;
  String draftBridgeUrl = '';
  String draftDomainSuffix = '';
  String draftDescription = '';

  Future<void> init() async {
    final p = await SharedPreferences.getInstance();
    baseUrl = (p.getString(_kBaseUrl) ?? baseUrl).trim();
    adminKey = (p.getString(_kAdminKey) ?? '').trim();
    nodeKey = (p.getString(_kNodeKey) ?? '').trim();
    inviteTtlSeconds = p.getInt(_kInviteTtl) ?? inviteTtlSeconds;
    inviteMaxUses = p.getInt(_kInviteMaxUses) ?? inviteMaxUses;

    draftPublic = p.getBool(_kDraftPublic) ?? draftPublic;
    draftHttpEnabled = p.getBool(_kDraftHttp) ?? draftHttpEnabled;
    draftHttpsEnabled = p.getBool(_kDraftHttps) ?? draftHttpsEnabled;
    draftBridgeUrl = (p.getString(_kDraftBridgeUrl) ?? draftBridgeUrl).trim();
    draftDomainSuffix = (p.getString(_kDraftDomainSuffix) ?? draftDomainSuffix)
        .trim();
    draftDescription = (p.getString(_kDraftDescription) ?? draftDescription)
        .trim();

    notifyListeners();
  }

  Future<void> persist() async {
    final p = await SharedPreferences.getInstance();
    await p.setString(_kBaseUrl, baseUrl.trim());
    await p.setString(_kAdminKey, adminKey.trim());
    await p.setString(_kNodeKey, nodeKey.trim());
    await p.setInt(_kInviteTtl, inviteTtlSeconds);
    await p.setInt(_kInviteMaxUses, inviteMaxUses);
  }

  Future<void> persistDraft() async {
    final p = await SharedPreferences.getInstance();
    await p.setBool(_kDraftPublic, draftPublic);
    await p.setBool(_kDraftHttp, draftHttpEnabled);
    await p.setBool(_kDraftHttps, draftHttpsEnabled);
    await p.setString(_kDraftBridgeUrl, draftBridgeUrl.trim());
    await p.setString(_kDraftDomainSuffix, draftDomainSuffix.trim());
    await p.setString(_kDraftDescription, draftDescription);
  }

  Future<void> updateDraft({
    required bool public,
    required bool httpEnabled,
    required bool httpsEnabled,
    required String bridgeUrl,
    required String domainSuffix,
    required String description,
    bool persistNow = false,
    bool notify = true,
  }) async {
    draftPublic = public;
    draftHttpEnabled = httpEnabled;
    draftHttpsEnabled = httpsEnabled;
    draftBridgeUrl = bridgeUrl.trim();
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
    this.baseUrl = baseUrl.trim();
    this.adminKey = adminKey.trim();
    this.nodeKey = nodeKey.trim();
    await persist();
    notifyListeners();
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
      draftBridgeUrl = cfg.bridgeUrl;
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
          'bridge_url': cfg.bridgeUrl,
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
    required String bridgeUrl,
  }) async {
    loading = true;
    notifyListeners();
    try {
      await updateDraft(
        public: public,
        httpEnabled: httpEnabled,
        httpsEnabled: httpsEnabled,
        bridgeUrl: bridgeUrl,
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
        bridgeUrl: bridgeUrl,
      );
      await refreshConfig();
      _setOutput({'code': 0, 'data': out});
    } catch (e) {
      _setOutput(_errToOut(e));
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
        ttlSeconds: inviteTtlSeconds,
        maxUses: inviteMaxUses,
        scopeJson: '{}',
      );
      lastInvite = res;
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
      _setOutput({'code': 0, 'data': out});
    } catch (e) {
      _setOutput(_errToOut(e));
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
