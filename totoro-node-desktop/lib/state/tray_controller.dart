import 'dart:async';
import 'dart:io' show Platform, exit;

import 'package:flutter/foundation.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:tray_manager/tray_manager.dart' hide MenuItem;
import 'package:tray_manager/tray_manager.dart' as tm;

import '../platform/system_metrics.dart';

// NOTE: tray_manager re-exports menu_base types.
import 'package:menu_base/menu_base.dart' as menu;

class TrayController with TrayListener {
  TrayController();

  Timer? _timer;
  NetworkBytes? _last;
  DateTime? _lastAt;

  String _speedLabel = '当前网速：--';

  Future<void> init() async {
    if (!(Platform.isMacOS || Platform.isWindows)) return;

    trayManager.addListener(this);

    // tray_manager expects assets path relative to flutter_assets.
    if (Platform.isWindows) {
      await trayManager.setIcon('assets/tray/tray.ico');
    } else {
      await trayManager.setIcon(
        'assets/tray/tray.png',
        isTemplate: true,
        iconSize: 18,
      );
    }
    await trayManager.setToolTip('Totoro');

    await _refreshMenu();

    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 1), (_) async {
      await _tick();
    });
  }

  Future<void> dispose() async {
    _timer?.cancel();
    _timer = null;
    trayManager.removeListener(this);
    try {
      await trayManager.destroy();
    } catch (_) {
      // ignore
    }
  }

  Future<void> _tick() async {
    try {
      final now = DateTime.now();
      final cur = await SystemMetrics.getNetworkBytes();
      if (cur == null) return;

      if (_last != null && _lastAt != null) {
        final dtMs = now.difference(_lastAt!).inMilliseconds;
        if (dtMs > 0) {
          final rxDelta = cur.rxBytes - _last!.rxBytes;
          final txDelta = cur.txBytes - _last!.txBytes;
          final sec = dtMs / 1000.0;
          final rxPerSec = rxDelta / sec;
          final txPerSec = txDelta / sec;
          _speedLabel =
              '当前网速：↓${_fmtRate(rxPerSec)}/s ↑${_fmtRate(txPerSec)}/s';
        }
      }
      _last = cur;
      _lastAt = now;

      await _refreshMenu();
    } catch (e) {
      if (kDebugMode) {
        // ignore noisy failures in production, but keep visible in debug
        // print(e);
      }
    }
  }

  Future<void> _refreshMenu() async {
    final items = <menu.MenuItem>[
      menu.MenuItem(key: 'speed', label: _speedLabel, disabled: true),
      menu.MenuItem.separator(),
      menu.MenuItem(key: 'quit', label: '退出'),
    ];
    final m = menu.Menu(items: items);
    await trayManager.setContextMenu(m);
  }

  @override
  void onTrayIconMouseDown() {
    trayManager.popUpContextMenu(bringAppToFront: false);
  }

  @override
  void onTrayIconRightMouseDown() {
    trayManager.popUpContextMenu(bringAppToFront: false);
  }

  @override
  void onTrayMenuItemClick(menu.MenuItem menuItem) {
    if (menuItem.key == 'quit') {
      // Ensure tray is removed; exit immediately.
      trayManager.destroy();
      exit(0);
    }
  }

  static String _fmtRate(double bytesPerSec) {
    final v = bytesPerSec.isFinite ? bytesPerSec : 0.0;
    const kb = 1024.0;
    const mb = 1024.0 * 1024.0;
    const gb = 1024.0 * 1024.0 * 1024.0;
    if (v >= gb) return '${(v / gb).toStringAsFixed(2)}GB';
    if (v >= mb) return '${(v / mb).toStringAsFixed(2)}MB';
    if (v >= kb) return '${(v / kb).toStringAsFixed(1)}KB';
    return '${v.toStringAsFixed(0)}B';
  }
}
