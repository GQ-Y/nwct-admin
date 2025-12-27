import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:flutter/services.dart' show rootBundle;
import 'package:http/http.dart' as http;

import '../state/app_controller.dart';

class NodeService {
  // 默认与当前后台节点保持一致（你现在用的是 18081）
  static const String _defaultBaseUrl = 'http://127.0.0.1:18081';
  static const String _defaultBridgeUrl = 'http://192.168.2.32:18090';

  static Process? _proc;
  static bool _startedByUs = false;

  static bool _looksLikeLocal(String baseUrl) {
    final b = baseUrl.trim();
    return b.startsWith('http://127.0.0.1') ||
        b.startsWith('http://localhost') ||
        b.startsWith('https://127.0.0.1') ||
        b.startsWith('https://localhost');
  }

  static String _randKey() {
    final r = Random.secure();
    final b = List<int>.generate(18, (_) => r.nextInt(256));
    return base64UrlEncode(b).replaceAll('=', '');
  }

  static Future<Directory> _appDataDir() async {
    if (Platform.isMacOS) {
      final home = Platform.environment['HOME'] ?? '';
      if (home.isNotEmpty) {
        return Directory('$home/Library/Application Support/totoro-node-desktop');
      }
    }
    if (Platform.isWindows) {
      final appData = Platform.environment['APPDATA'] ?? '';
      if (appData.isNotEmpty) {
        return Directory('$appData\\totoro-node-desktop');
      }
    }
    // fallback
    return Directory('${Directory.current.path}${Platform.pathSeparator}totoro-node-desktop-data');
  }

  static Future<bool> _ping(String baseUrl, {required String adminKey}) async {
    final u = Uri.parse('${baseUrl.replaceAll(RegExp(r'/*$'), '')}/api/v1/node/config');
    try {
      final res = await http
          .get(u, headers: adminKey.isEmpty ? {} : {'X-Admin-Key': adminKey})
          .timeout(const Duration(milliseconds: 700));
      return res.statusCode >= 200 && res.statusCode < 300;
    } catch (_) {
      return false;
    }
  }

  static Future<File> _materializeAsset({
    required String assetPath,
    required File target,
    bool executable = false,
  }) async {
    final data = await rootBundle.load(assetPath);
    final bytes = data.buffer.asUint8List();
    await target.parent.create(recursive: true);
    await target.writeAsBytes(bytes, flush: true);
    if (executable && !Platform.isWindows) {
      try {
        await Process.run('chmod', ['+x', target.path]);
      } catch (_) {}
    }
    return target;
  }

  static Future<void> ensureStarted(AppController c) async {
    // 只对“本机 node”启用自动拉起；如果用户配置了远程 baseUrl，就不干预。
    if (!_looksLikeLocal(c.baseUrl)) return;

    // 如果已经通了（可能是用户手动启动的 node），直接复用。
    final ak = c.adminKey.trim();
    final desiredBaseUrl = c.baseUrl.trim().isNotEmpty ? c.baseUrl.trim() : _defaultBaseUrl;
    if (await _ping(desiredBaseUrl, adminKey: ak)) {
      return;
    }

    final adminKey = ak.isNotEmpty ? ak : _randKey();
    final baseUrl = desiredBaseUrl;
    final uri = Uri.tryParse(baseUrl);
    final port = (uri != null && uri.hasPort) ? uri.port : 18081;

    final dir = await _appDataDir();
    await dir.create(recursive: true);

    final binName = Platform.isWindows ? 'totoro-node.exe' : 'totoro-node';
    final binFile = File('${dir.path}${Platform.pathSeparator}$binName');
    final frpsFile = File('${dir.path}${Platform.pathSeparator}frps.toml');
    final dbFile = File('${dir.path}${Platform.pathSeparator}node.db');

    // 资源准备（首次解压 / 覆盖更新）
    if (Platform.isMacOS) {
      await _materializeAsset(
        assetPath: 'assets/node/totoro-node-macos',
        target: binFile,
        executable: true,
      );
    } else if (Platform.isWindows) {
      await _materializeAsset(
        assetPath: 'assets/node/totoro-node-windows.exe',
        target: binFile,
        executable: true,
      );
    } else {
      // 其它平台暂不内置拉起
      return;
    }
    if (!await frpsFile.exists()) {
      await _materializeAsset(assetPath: 'assets/node/frps.toml', target: frpsFile);
    }

    // 端口占用：如果 baseUrl 端口已被占用，但 ping 不通，说明不是我们的 node；避免硬拉起冲突
    // 简化处理：直接尝试拉起，失败由探活兜底。

    final env = <String, String>{
      'TOTOTO_NODE_API_ADDR': ':$port',
      'TOTOTO_FRPS_CONFIG': frpsFile.path,
      'TOTOTO_NODE_DB': dbFile.path,
      'TOTOTO_NODE_ADMIN_KEY': adminKey,
      'TOTOTO_BRIDGE_URL': _defaultBridgeUrl,
    };

    final logFile = File('${dir.path}${Platform.pathSeparator}totoro-node.log');
    final logSink = logFile.openWrite(mode: FileMode.append);

    try {
      _proc = await Process.start(
        binFile.path,
        const [],
        workingDirectory: dir.path,
        environment: env,
        runInShell: false,
      );
      _startedByUs = true;
      _proc!.stdout.transform(utf8.decoder).listen((s) => logSink.write(s));
      _proc!.stderr.transform(utf8.decoder).listen((s) => logSink.write(s));
      _proc!.exitCode.whenComplete(() => logSink.flush().whenComplete(() => logSink.close()));
    } catch (_) {
      await logSink.flush();
      await logSink.close();
      return;
    }

    // 等待 node API 就绪
    final deadline = DateTime.now().add(const Duration(seconds: 8));
    while (DateTime.now().isBefore(deadline)) {
      if (await _ping(baseUrl, adminKey: adminKey)) {
        await c.updateConnection(baseUrl: baseUrl, adminKey: adminKey, nodeKey: '');
        return;
      }
      await Future<void>.delayed(const Duration(milliseconds: 250));
    }
  }

  static Future<void> stopIfStartedByUs() async {
    if (!_startedByUs) return;
    try {
      _proc?.kill(ProcessSignal.sigterm);
    } catch (_) {}
    _startedByUs = false;
    _proc = null;
  }
}


