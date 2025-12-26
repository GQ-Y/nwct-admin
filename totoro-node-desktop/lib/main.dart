import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

void main() {
  runApp(const MainApp());
}

class MainApp extends StatelessWidget {
  const MainApp({super.key});

  @override
  Widget build(BuildContext context) {
    return const MaterialApp(
      debugShowCheckedModeBanner: false,
      home: NodeAdminPage(),
    );
  }
}

class NodeAdminPage extends StatefulWidget {
  const NodeAdminPage({super.key});

  @override
  State<NodeAdminPage> createState() => _NodeAdminPageState();
}

class _NodeAdminPageState extends State<NodeAdminPage> {
  final _baseUrl = TextEditingController(text: 'http://127.0.0.1:18080');
  final _adminKey = TextEditingController();
  final _nodeKey = TextEditingController(text: 'change_me');

  bool _public = false;
  final _bridgeUrl = TextEditingController(text: 'http://127.0.0.1:18090');
  final _ttl = TextEditingController(text: '86400');
  final _maxUses = TextEditingController(text: '50');

  bool _loading = false;
  String _out = 'ready';

  Map<String, String> _headers({bool includeNodeKey = false}) {
    final h = <String, String>{'Content-Type': 'application/json'};
    final admin = _adminKey.text.trim();
    if (admin.isNotEmpty) h['X-Admin-Key'] = admin;
    if (includeNodeKey) {
      final nk = _nodeKey.text.trim();
      if (nk.isNotEmpty) h['X-Node-Key'] = nk;
    }
    return h;
  }

  String _url(String path) {
    final b = _baseUrl.text.trim().replaceAll(RegExp(r'/*$'), '');
    if (path.startsWith('/')) return '$b$path';
    return '$b/$path';
  }

  void _setOut(dynamic v) {
    setState(() {
      if (v is String) {
        _out = v;
      } else {
        _out = const JsonEncoder.withIndent('  ').convert(v);
      }
    });
  }

  Future<void> _loadConfig() async {
    setState(() => _loading = true);
    try {
      final res = await http.get(Uri.parse(_url('/api/v1/node/config')), headers: _headers());
      final j = jsonDecode(res.body);
      _setOut(j);
      if (j is Map && j['code'] == 0 && j['data'] is Map) {
        final d = j['data'] as Map;
        setState(() {
          _public = d['public'] == true;
          _bridgeUrl.text = (d['bridge_url'] ?? '').toString();
        });
      }
    } catch (e) {
      _setOut('error: $e');
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _saveConfig() async {
    setState(() => _loading = true);
    try {
      final body = jsonEncode({'public': _public, 'bridge_url': _bridgeUrl.text.trim()});
      final res = await http.post(Uri.parse(_url('/api/v1/node/config')), headers: _headers(includeNodeKey: true), body: body);
      final j = jsonDecode(res.body);
      _setOut(j);
    } catch (e) {
      _setOut('error: $e');
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _createInvite() async {
    setState(() => _loading = true);
    try {
      final ttl = int.tryParse(_ttl.text.trim()) ?? 0;
      final maxUses = int.tryParse(_maxUses.text.trim()) ?? 0;
      final body = jsonEncode({'ttl_seconds': ttl, 'max_uses': maxUses, 'scope_json': '{}'});
      final res = await http.post(Uri.parse(_url('/api/v1/node/invites')), headers: _headers(), body: body);
      final j = jsonDecode(res.body);
      _setOut(j);
    } catch (e) {
      _setOut('error: $e');
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  void dispose() {
    _baseUrl.dispose();
    _adminKey.dispose();
    _nodeKey.dispose();
    _bridgeUrl.dispose();
    _ttl.dispose();
    _maxUses.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Totoro Node Desktop'),
        actions: [
          TextButton(
            onPressed: _loading ? null : _loadConfig,
            child: const Text('刷新'),
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            SizedBox(
              width: 420,
              child: ListView(
                children: [
                  const Text('连接设置', style: TextStyle(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 10),
                  TextField(controller: _baseUrl, decoration: const InputDecoration(labelText: 'Node API Base URL')),
                  TextField(controller: _adminKey, decoration: const InputDecoration(labelText: 'X-Admin-Key（可选）')),
                  TextField(controller: _nodeKey, decoration: const InputDecoration(labelText: 'X-Node-Key（更新配置需要）')),
                  const SizedBox(height: 18),
                  const Divider(),
                  SwitchListTile(
                    value: _public,
                    onChanged: _loading ? null : (v) => setState(() => _public = v),
                    title: const Text('公开节点（public）'),
                  ),
                  TextField(controller: _bridgeUrl, decoration: const InputDecoration(labelText: 'Bridge URL')),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      Expanded(
                        child: ElevatedButton(
                          onPressed: _loading ? null : _saveConfig,
                          child: const Text('保存配置'),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 18),
                  const Divider(),
                  const Text('邀请码', style: TextStyle(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      Expanded(
                        child: TextField(controller: _ttl, decoration: const InputDecoration(labelText: 'TTL(s)')),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: TextField(controller: _maxUses, decoration: const InputDecoration(labelText: '次数')),
                      ),
                    ],
                  ),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      Expanded(
                        child: ElevatedButton(
                          onPressed: _loading ? null : _createInvite,
                          child: const Text('生成邀请码'),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 18),
                  if (_loading) const LinearProgressIndicator(),
                ],
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Container(
                decoration: BoxDecoration(
                  color: const Color(0xFF0B1020),
                  borderRadius: BorderRadius.circular(12),
                ),
                padding: const EdgeInsets.all(12),
                child: SingleChildScrollView(
                  child: SelectableText(
                    _out,
                    style: const TextStyle(fontFamily: 'monospace', fontSize: 12, color: Color(0xFFE5E7EB)),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
