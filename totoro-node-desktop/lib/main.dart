import 'package:flutter/material.dart';

import 'dart:io' show Platform;

import 'package:bitsdojo_window/bitsdojo_window.dart';

import 'app.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  if (Platform.isWindows) {
    doWhenWindowReady(() {
      // Keep consistent with macOS minSize, and ensure layouts never get cramped.
      appWindow.minSize = const Size(1040, 700);
      appWindow.size = const Size(1280, 720);
      appWindow.alignment = Alignment.center;
      appWindow.show();
    });
  }
  runApp(const TotoroNodeDesktopApp());
}
