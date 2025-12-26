import 'dart:convert';

import 'package:http/http.dart' as http;

class ApiException implements Exception {
  ApiException({required this.message, this.httpStatus, this.code});

  final String message;
  final int? httpStatus;
  final int? code;

  @override
  String toString() {
    final parts = <String>[];
    if (httpStatus != null) parts.add('http=$httpStatus');
    if (code != null) parts.add('code=$code');
    final meta = parts.isEmpty ? '' : ' (${parts.join(', ')})';
    return 'ApiException: $message$meta';
  }
}

class NodeConfig {
  NodeConfig({
    required this.public,
    required this.description,
    required this.domainSuffix,
    required this.httpEnabled,
    required this.httpsEnabled,
    required this.bridgeUrl,
  });

  final bool public;
  final String description;
  final String domainSuffix;
  final bool httpEnabled;
  final bool httpsEnabled;
  final String bridgeUrl;

  factory NodeConfig.fromJson(Map<String, dynamic> j) {
    return NodeConfig(
      public: j['public'] == true,
      description: (j['description'] ?? '').toString(),
      domainSuffix: (j['domain_suffix'] ?? '').toString(),
      httpEnabled: j['http_enabled'] == true,
      httpsEnabled: j['https_enabled'] == true,
      bridgeUrl: (j['bridge_url'] ?? '').toString(),
    );
  }

  Map<String, dynamic> toPatchJson({
    required bool public,
    required bool httpEnabled,
    required bool httpsEnabled,
    required String description,
    required String domainSuffix,
    required String bridgeUrl,
  }) {
    return <String, dynamic>{
      'public': public,
      'description': description,
      'domain_suffix': domainSuffix,
      'http_enabled': httpEnabled,
      'https_enabled': httpsEnabled,
      'bridge_url': bridgeUrl,
    };
  }
}

class CreateInviteResult {
  CreateInviteResult({
    required this.inviteId,
    required this.code,
    required this.expiresAt,
  });

  final String inviteId;
  final String code;
  final String expiresAt;

  factory CreateInviteResult.fromJson(Map<String, dynamic> j) {
    return CreateInviteResult(
      inviteId: (j['invite_id'] ?? '').toString(),
      code: (j['code'] ?? '').toString(),
      expiresAt: (j['expires_at'] ?? '').toString(),
    );
  }
}

class NodeApiClient {
  NodeApiClient({
    required this.baseUrl,
    required this.adminKey,
    required this.nodeKey,
  });

  final String baseUrl;
  final String adminKey;
  final String nodeKey;

  Uri _uri(String path) {
    final b = baseUrl.trim().replaceAll(RegExp(r'/*$'), '');
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$b$p');
  }

  Map<String, String> _headers({bool includeNodeKey = false}) {
    final h = <String, String>{'Content-Type': 'application/json'};
    final ak = adminKey.trim();
    if (ak.isNotEmpty) h['X-Admin-Key'] = ak;
    if (includeNodeKey) {
      final nk = nodeKey.trim();
      if (nk.isNotEmpty) h['X-Node-Key'] = nk;
    }
    return h;
  }

  dynamic _decodeBody(http.Response res) {
    final txt = res.body;
    dynamic j;
    try {
      j = txt.isEmpty ? null : jsonDecode(txt);
    } catch (_) {
      throw ApiException(
        message: txt.isNotEmpty ? txt : 'HTTP ${res.statusCode}',
        httpStatus: res.statusCode,
      );
    }
    if (j is Map<String, dynamic>) return j;
    if (j is Map) return j.map((k, v) => MapEntry(k.toString(), v));
    throw ApiException(message: '响应格式错误', httpStatus: res.statusCode);
  }

  T _unwrapData<T>(http.Response res) {
    final j = _decodeBody(res);
    if (j is! Map<String, dynamic>) {
      throw ApiException(message: '响应格式错误', httpStatus: res.statusCode);
    }
    final code = j['code'];
    final msg = (j['message'] ?? '').toString();
    final ok = code is int ? code == 0 : true;
    final httpOk = res.statusCode >= 200 && res.statusCode < 300;
    if (!httpOk || !ok) {
      throw ApiException(
        message: msg.isNotEmpty ? msg : '请求失败',
        httpStatus: res.statusCode,
        code: code is int ? code : null,
      );
    }
    return j['data'] as T;
  }

  Future<NodeConfig> getConfig() async {
    final res = await http.get(
      _uri('/api/v1/node/config'),
      headers: _headers(),
    );
    final data = _unwrapData<dynamic>(res);
    if (data is Map) {
      return NodeConfig.fromJson(data.map((k, v) => MapEntry(k.toString(), v)));
    }
    throw ApiException(message: '配置响应格式错误', httpStatus: res.statusCode);
  }

  Future<Map<String, dynamic>> updateConfig({
    required bool public,
    required bool httpEnabled,
    required bool httpsEnabled,
    required String description,
    required String domainSuffix,
    required String bridgeUrl,
  }) async {
    final res = await http.post(
      _uri('/api/v1/node/config'),
      headers: _headers(includeNodeKey: true),
      body: jsonEncode({
        'public': public,
        'description': description,
        'domain_suffix': domainSuffix,
        'http_enabled': httpEnabled,
        'https_enabled': httpsEnabled,
        'bridge_url': bridgeUrl,
      }),
    );
    final data = _unwrapData<dynamic>(res);
    if (data is Map) {
      return data.map((k, v) => MapEntry(k.toString(), v));
    }
    return <String, dynamic>{'ok': true};
  }

  void _requireNodeKey() {
    if (nodeKey.trim().isEmpty) {
      throw ApiException(message: '缺少 X-Node-Key（敏感操作需要）');
    }
  }

  Future<CreateInviteResult> createInvite({
    required int ttlSeconds,
    required int maxUses,
    required String scopeJson,
  }) async {
    _requireNodeKey();
    final res = await http.post(
      _uri('/api/v1/node/invites'),
      headers: _headers(includeNodeKey: true),
      body: jsonEncode({
        'ttl_seconds': ttlSeconds,
        'max_uses': maxUses,
        'scope_json': scopeJson,
      }),
    );
    final data = _unwrapData<dynamic>(res);
    if (data is Map) {
      return CreateInviteResult.fromJson(
        data.map((k, v) => MapEntry(k.toString(), v)),
      );
    }
    throw ApiException(message: '生成邀请码响应格式错误', httpStatus: res.statusCode);
  }

  Future<Map<String, dynamic>> revokeInvite({required String inviteId}) async {
    _requireNodeKey();
    final id = inviteId.trim();
    if (id.isEmpty) throw ApiException(message: 'invite_id 不能为空');
    final res = await http.post(
      _uri('/api/v1/node/invites/revoke'),
      headers: _headers(includeNodeKey: true),
      body: jsonEncode({'invite_id': id}),
    );
    final data = _unwrapData<dynamic>(res);
    if (data is Map) {
      return data.map((k, v) => MapEntry(k.toString(), v));
    }
    return <String, dynamic>{'revoked': true};
  }
}
