import 'dart:io' show Platform;

import 'package:flutter/services.dart';

class NetworkBytes {
  NetworkBytes({required this.rxBytes, required this.txBytes});

  final int rxBytes;
  final int txBytes;
}

class SystemMetrics {
  static const MethodChannel _ch = MethodChannel('totoro/system');

  static Future<NetworkBytes?> getNetworkBytes() async {
    if (!(Platform.isMacOS || Platform.isWindows)) return null;
    final res = await _ch.invokeMethod<dynamic>('getNetworkBytes');
    if (res is Map) {
      final m = res.map((k, v) => MapEntry(k.toString(), v));
      final rx = (m['rx'] is int)
          ? (m['rx'] as int)
          : int.tryParse('${m['rx']}');
      final tx = (m['tx'] is int)
          ? (m['tx'] as int)
          : int.tryParse('${m['tx']}');
      if (rx != null && tx != null)
        return NetworkBytes(rxBytes: rx, txBytes: tx);
    }
    return null;
  }
}
